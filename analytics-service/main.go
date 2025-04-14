package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// DB-соединение
var db *sql.DB

// initDB устанавливает соединение с базой данных analytics-db.
// Переменные окружения: DB_HOST, DB_USER, DB_PASSWORD, DB_NAME должны быть заданы.
func initDB() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := "5432" // В контейнере PostgreSQL слушает на 5432

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

	// Инициализация роутера Gin
	r := gin.Default()

	// Health-check endpoint.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Эндпоинт для получения агрегированных данных аналитики.
	// В данном примере производится выборка последней строки из таблицы order_stats.
	r.GET("/analytics/stats", getAnalyticsStats)

	// Запуск сервиса на порту 8080
	r.Run(":8080")
}

// getAnalyticsStats выбирает последнюю запись статистики из таблицы order_stats.
// Предполагается, что таблица order_stats имеет следующую схему:
//   id SERIAL PRIMARY KEY,
//   total_orders INTEGER,
//   new_orders INTEGER,
//   completed_orders INTEGER,
//   calculated_at TIMESTAMP NOT NULL
func getAnalyticsStats(c *gin.Context) {
	var total, newOrders, completed int

	query := "SELECT total_orders, new_orders, completed_orders FROM order_stats ORDER BY calculated_at DESC LIMIT 1"
	err := db.QueryRow(query).Scan(&total, &newOrders, &completed)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Данные аналитики не найдены"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"total_orders":     total,
		"new_orders":       newOrders,
		"completed_orders": completed,
	})
}
