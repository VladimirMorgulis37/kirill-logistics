package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
	"net/smtp"

	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
	_ "github.com/lib/pq"
)

// Notification описывает уведомление.
type Notification struct {
	ID        int       `json:"id"`
	Type      string    `json:"type"`      // Тип уведомления (например, "order_completed")
	Recipient string    `json:"recipient"` // Email или телефон получателя
	Message   string    `json:"message"`   // Текст уведомления
	Status    string    `json:"status"`    // Статус ("pending", "sent", "failed")
	CreatedAt time.Time `json:"created_at"`
}

// NotificationMessage описывает событие, которое публикуется в RabbitMQ.
type NotificationMessage struct {
	Type      string `json:"type"`
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
}

var db *sql.DB

// initDB устанавливает подключение к базе данных уведомлений.
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

// connectToRabbitMQ устанавливает соединение с RabbitMQ.
func connectToRabbitMQ() (*amqp.Connection, error) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	return amqp.Dial(rabbitURL)
}

// startConsumer подписывается на очередь уведомлений в RabbitMQ и обрабатывает входящие сообщения.
func startConsumer() {
	conn, err := connectToRabbitMQ()
	if err != nil {
		log.Fatalf("Не удалось подключиться к RabbitMQ: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Не удалось открыть канал RabbitMQ: %v", err)
	}

	q, err := ch.QueueDeclare(
		"notifications", // имя очереди
		true,            // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		log.Fatalf("Ошибка объявления очереди: %v", err)
	}

	msgs, err := ch.Consume(
		q.Name, // очередь
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("Ошибка регистрации потребителя: %v", err)
	}

	log.Println(" [*] Ожидание сообщений из RabbitMQ")
	go func() {
		for d := range msgs {
			var notifMsg NotificationMessage
			if err := json.Unmarshal(d.Body, &notifMsg); err != nil {
				log.Printf("Ошибка декодирования сообщения: %v", err)
				continue
			}
			log.Printf("Получено сообщение: %+v", notifMsg)
			// Обработка уведомления: сохранить в БД и/или вызвать API для отправки SMS/email
			if err := processNotification(notifMsg); err != nil {
				log.Printf("Ошибка обработки уведомления: %v", err)
			}
		}
	}()
}

func sendEmail(to, subject, body string) error {
	from := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASS")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	addr := fmt.Sprintf("%s:%s", host, port)

	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-version: 1.0;\r\nContent-Type: text/plain; charset=\"UTF-8\";\r\n\r\n" +
		body + "\r\n")

	auth := smtp.PlainAuth("", from, password, host)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
// processNotification сохраняет уведомление в БД и имитирует отправку уведомления.
func processNotification(msg NotificationMessage) error {
	// Создаем уведомление со статусом "pending"
	n := Notification{
		Type:      msg.Type,
		Recipient: msg.Recipient,
		Message:   msg.Message,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	query := `
			INSERT INTO notifications (type, recipient, message, status, created_at)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`
	err := db.QueryRow(query, n.Type, n.Recipient, n.Message, n.Status, n.CreatedAt).Scan(&n.ID)
	if err != nil {
		return err
	}
	err = sendEmail(n.Recipient, "Уведомление от службы доставки", n.Message)
	if err != nil {
		log.Printf("Ошибка отправки письма: %v", err)
		_, _ = db.Exec("UPDATE notifications SET status = $1 WHERE id = $2", "failed", n.ID)
		return err
	}

	_, err = db.Exec("UPDATE notifications SET status = $1 WHERE id = $2", "sent", n.ID)
	if err != nil {
		return err
	}
	log.Printf("Уведомление сохранено и отправлено, ID: %d", n.ID)
	return nil
}

func checkRabbitMQ() string {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Printf("Не удалось подключиться к RabbitMQ: %v", err)
		return "unavailable"
	}
	conn.Close()
	return "ok"
}

func main() {
	var err error
	db, err = initDB()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД уведомлений: %v", err)
	}

	// Запускаем получение уведомлений из RabbitMQ в фоне
	startConsumer()

	// Инициализируем HTTP API Notification Service.
	r := gin.Default()
	// Эндпоинт для проверки работоспособности (health-check)
	r.GET("/health", func(c *gin.Context) {
		// Проверка БД
		dbStatus := "ok"
		if err := db.Ping(); err != nil {
			log.Printf("База данных недоступна: %v", err)
			dbStatus = "unavailable"
		}
		// Проверка RabbitMQ
		rabbitStatus := checkRabbitMQ()

		// Если база данных недоступна, считаем сервис «degraded» (или unhealthy)
		overallStatus := "ok"
		if dbStatus != "ok" {
			overallStatus = "degraded"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    overallStatus,
			"db":        dbStatus,
			"rabbitmq":  rabbitStatus,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Эндпоинт для получения списка уведомлений
	r.GET("/notifications", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, type, recipient, message, status, created_at FROM notifications ORDER BY created_at DESC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		var notifications []Notification
		for rows.Next() {
			var n Notification
			if err := rows.Scan(&n.ID, &n.Type, &n.Recipient, &n.Message, &n.Status, &n.CreatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			notifications = append(notifications, n)
		}
		c.JSON(http.StatusOK, notifications)
	})
	// Эндпоинт для получения уведомления по ID
	r.GET("/notifications/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}
		var n Notification
		query := "SELECT id, type, recipient, message, status, created_at FROM notifications WHERE id = $1"
		err = db.QueryRow(query, id).Scan(&n.ID, &n.Type, &n.Recipient, &n.Message, &n.Status, &n.CreatedAt)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Уведомление не найдено"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(http.StatusOK, n)
	})

	r.Run(":8080") // Notification Service слушает на порту 8080
}
