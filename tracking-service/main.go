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

// TrackingInfo описывает структуру записи трекинга.
type TrackingInfo struct {
	OrderID   string    `json:"order_id"`   // Идентификатор заказа (PRIMARY KEY)
	CourierID string    `json:"courier_id"` // Идентификатор курьера, привязанного к заказу
	Status    string    `json:"status"`     // Статус доставки (например, "в пути", "доставлен")
	UpdatedAt time.Time `json:"updated_at"` // Время последнего обновления статуса
}

// Глобальное подключение к базе данных.
var db *sql.DB

// initDB устанавливает соединение с PostgreSQL, используя переменные окружения:
// DB_HOST, DB_USER, DB_PASSWORD, DB_NAME.
func initDB() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := "5432" // PostgreSQL внутри контейнера слушает на 5432

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

	// Инициализируем роутер Gin.
	r := gin.Default()

	// Health-check endpoint.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Endpoint для обновления данных трекинга (метод POST).
	// Если запись по order_id уже существует, будет выполнен UPDATE (использование UPSERT).
	r.POST("/tracking", func(c *gin.Context) {
		var t TrackingInfo
		if err := c.BindJSON(&t); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
			return
		}
		t.UpdatedAt = time.Now()
		query := `
			INSERT INTO tracking_info (order_id, courier_id, status, updated_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (order_id) DO UPDATE SET
				courier_id = EXCLUDED.courier_id,
				status = EXCLUDED.status,
				updated_at = EXCLUDED.updated_at;
		`
		_, err := db.Exec(query, t.OrderID, t.CourierID, t.Status, t.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, t)
	})

	// Endpoint для получения информации трекинга по order_id (метод GET).
	r.GET("/tracking/:orderId", func(c *gin.Context) {
		orderId := c.Param("orderId")
		var t TrackingInfo
		query := "SELECT order_id, courier_id, status, updated_at FROM tracking_info WHERE order_id = $1"
		err := db.QueryRow(query, orderId).Scan(&t.OrderID, &t.CourierID, &t.Status, &t.UpdatedAt)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Информация трекинга не найдена"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, t)
	})

	// Запускаем Tracking Service на порту 8080.
	r.Run(":8080")
}
