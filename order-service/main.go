package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
)

// Order описывает заказ с расширенными полями для физический характеристик.
type Order struct {
	ID            string    `json:"id"`
	SenderName    string    `json:"sender_name"`
	RecipientName string    `json:"recipient_name"`
	AddressFrom   string    `json:"address_from"`
	AddressTo     string    `json:"address_to"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`

	// Новые поля для расчёта доставки
	Weight  float64 `json:"weight"`   // вес посылки (кг)
	Length  float64 `json:"length"`   // длина (метры)
	Width   float64 `json:"width"`    // ширина (метры)
	Height  float64 `json:"height"`   // высота (метры)
	Urgency int     `json:"urgency"`  // 1 - стандартная, 2 - экспресс
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
	corsConfig := cors.Config{
        AllowOrigins:     []string{"http://localhost:3000"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }
    r.Use(cors.New(corsConfig))
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
		query := `
			INSERT INTO orders
				(id, sender_name, recipient_name, address_from, address_to, status, created_at, weight, length, width, height, urgency)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`
		_, err := db.Exec(query,
			o.ID,
			o.SenderName,
			o.RecipientName,
			o.AddressFrom,
			o.AddressTo,
			o.Status,
			o.CreatedAt,
			o.Weight,
			o.Length,
			o.Width,
			o.Height,
			o.Urgency,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, o)
	})

	r.GET("/orders", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, sender_name, recipient_name, address_from, address_to, status, created_at, weight, length, width, height, urgency FROM orders")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		var orders []Order
		for rows.Next() {
			var o Order
			if err := rows.Scan(&o.ID, &o.SenderName, &o.RecipientName, &o.AddressFrom, &o.AddressTo, &o.Status, &o.CreatedAt, &o.Weight, &o.Length, &o.Width, &o.Height, &o.Urgency); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			orders = append(orders, o)
		}
		c.JSON(http.StatusOK, orders)
	})

	r.GET("/orders/:id", func(c *gin.Context) {
		id := c.Param("id")
		var o Order
		query := "SELECT id, sender_name, recipient_name, address_from, address_to, status, created_at, weight, length, width, height, urgency FROM orders WHERE id = $1"
		err := db.QueryRow(query, id).Scan(&o.ID, &o.SenderName, &o.RecipientName, &o.AddressFrom, &o.AddressTo, &o.Status, &o.CreatedAt, &o.Weight, &o.Length, &o.Width, &o.Height, &o.Urgency)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Заказ не найден"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, o)
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
