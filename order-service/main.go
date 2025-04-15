package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
	_ "github.com/lib/pq"
)

// Order описывает заказ.
type Order struct {
	ID            string    `json:"id"`
	SenderName    string    `json:"sender_name"`
	RecipientName string    `json:"recipient_name"`
	AddressFrom   string    `json:"address_from"`
	AddressTo     string    `json:"address_to"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

var db *sql.DB

// Рекомендуется создать отдельную функцию для подключения к БД.
func initDB() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := "5432"

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	database, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}
	if err = database.Ping(); err != nil {
		return nil, err
	}
	return database, nil
}

// publishOrderCompletedEvent публикует событие завершённого заказа в RabbitMQ.
func publishOrderCompletedEvent(orderID string) error {
	rabbitURL := os.Getenv("RABBITMQ_URL") // Например: "amqp://user:password@rabbitmq:5672/"
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return fmt.Errorf("dial: %s", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("channel: %s", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"order_completed", // имя очереди
		true,              // durable
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("queue declare: %s", err)
	}

	// Формируем сообщение с событием завершения заказа.
	event := map[string]string{
		"order_id": orderID,
		"event":    "order_completed",
		"message":  "Заказ успешно завершён и доставлен",
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json marshal: %s", err)
	}

	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		return fmt.Errorf("publish: %s", err)
	}
	return nil
}

func main() {
	var err error
	db, err = initDB()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	r := gin.Default()

	// Health-check endpoint.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Endpoint для создания заказа.
	r.POST("/orders", func(c *gin.Context) {
		var o Order
		if err := c.BindJSON(&o); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
			return
		}
		o.ID = time.Now().Format("20060102150405")
		o.CreatedAt = time.Now()
		o.Status = "новый"
		query := `INSERT INTO orders (id, sender_name, recipient_name, address_from, address_to, status, created_at)
		          VALUES ($1, $2, $3, $4, $5, $6, $7)`
		_, err := db.Exec(query, o.ID, o.SenderName, o.RecipientName, o.AddressFrom, o.AddressTo, o.Status, o.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, o)
	})

	// Endpoint для завершения заказа: Курьер нажимает "Завершить заказ".
	r.PUT("/orders/:id/finish", func(c *gin.Context) {
		orderID := c.Param("id")

		// Обновляем статус заказа на "завершённый".
		query := "UPDATE orders SET status = $1 WHERE id = $2"
		_, err := db.Exec(query, "завершён", orderID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Публикуем событие в RabbitMQ, чтобы уведомить Notification Service.
		if err := publishOrderCompletedEvent(orderID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка отправки уведомления: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "Заказ завершён и уведомление отправлено"})
	})

	// (Дополнительно можно добавить GET /orders/:id и другие CRUD-эндпоинты)

	r.Run(":8080")
}
