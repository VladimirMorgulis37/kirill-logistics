package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// Order представляет структуру заказа для логистической системы.
type Order struct {
	ID            string    `json:"id"`
	SenderName    string    `json:"sender_name"`
	RecipientName string    `json:"recipient_name"`
	AddressFrom   string    `json:"address_from"`
	AddressTo     string    `json:"address_to"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// Глобальная переменная для подключения к базе данных.
var db *sql.DB

// initDB устанавливает соединение с базой данных order-db.
// Переменные окружения: DB_HOST, DB_USER, DB_PASSWORD, DB_NAME
func initDB() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := "5432" // PostgreSQL внутри контейнера всегда слушает на 5432.

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

func main() {
	var err error
	db, err = initDB()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	// Инициализируем маршруты с использованием Gin.
	r := gin.Default()

	// Health-check endpoint.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Создание заказа: POST /orders.
	r.POST("/orders", func(c *gin.Context) {
		var o Order
		if err := c.BindJSON(&o); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
			return
		}
		// Генерируем ID на основе времени (можно использовать UUID для production).
		o.ID = time.Now().Format("20060102150405")
		o.CreatedAt = time.Now()
		o.Status = "новый" // начальный статус заказа.

		query := `INSERT INTO orders (id, sender_name, recipient_name, address_from, address_to, status, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)`
		_, err := db.Exec(query, o.ID, o.SenderName, o.RecipientName, o.AddressFrom, o.AddressTo, o.Status, o.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, o)
	})

	// Получение заказа по ID: GET /orders/:id.
	r.GET("/orders/:id", func(c *gin.Context) {
		id := c.Param("id")
		var o Order
		query := "SELECT id, sender_name, recipient_name, address_from, address_to, status, created_at FROM orders WHERE id = $1"
		err := db.QueryRow(query, id).Scan(&o.ID, &o.SenderName, &o.RecipientName, &o.AddressFrom, &o.AddressTo, &o.Status, &o.CreatedAt)
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

	// Получение списка заказов: GET /orders.
	r.GET("/orders", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, sender_name, recipient_name, address_from, address_to, status, created_at FROM orders")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		orders := []Order{}
		for rows.Next() {
			var o Order
			if err := rows.Scan(&o.ID, &o.SenderName, &o.RecipientName, &o.AddressFrom, &o.AddressTo, &o.Status, &o.CreatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			orders = append(orders, o)
		}
		c.JSON(http.StatusOK, orders)
	})

	// Обновление заказа: PUT /orders/:id.
	r.PUT("/orders/:id", func(c *gin.Context) {
		id := c.Param("id")
		var o Order
		if err := c.BindJSON(&o); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
			return
		}
		query := `UPDATE orders SET sender_name = $1, recipient_name = $2, address_from = $3, address_to = $4, status = $5, created_at = $6 WHERE id = $7`
		_, err := db.Exec(query, o.SenderName, o.RecipientName, o.AddressFrom, o.AddressTo, o.Status, o.CreatedAt, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		o.ID = id
		c.JSON(http.StatusOK, o)
	})

	// Удаление заказа: DELETE /orders/:id.
	r.DELETE("/orders/:id", func(c *gin.Context) {
		id := c.Param("id")
		result, err := db.Exec("DELETE FROM orders WHERE id = $1", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil || rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Заказ не найден"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "Заказ удалён"})
	})

	// Запускаем сервис на порту 8080.
	r.Run(":8080")
}
