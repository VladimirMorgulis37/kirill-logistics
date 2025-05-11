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

var db *sql.DB

// initDB устанавливает соединение с базой данных аналитики.
func initDB() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := "5432"

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func main() {
	var err error
	db, err = initDB()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	r := gin.Default()

	// Health-check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Общая статистика заказов с фильтром по периоду
	r.GET("/analytics/general", getGeneralStats)

	if err := r.Run(":8081"); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

// GeneralStatsResponse отвечает общей статистикой заказов.
type GeneralStatsResponse struct {
	TotalOrders               int     `json:"total_orders"`
	ActiveOrders              int     `json:"active_orders"`
	CompletedOrders           int     `json:"completed_orders"`
	AverageCompletionTimeSecs float64 `json:"average_completion_time_seconds"`
}

// getGeneralStats вычисляет метрики по заказам за выбранный период.
func getGeneralStats(c *gin.Context) {
	// Парсим параметры периода
	fromParam := c.Query("from")
	toParam := c.Query("to")
	var fromTime, toTime time.Time
	var err error
	if fromParam != "" {
		fromTime, err = time.Parse("2006-01-02", fromParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from date format, use YYYY-MM-DD"})
			return
		}
	} else {
		fromTime = time.Time{} // минимум
	}
	if toParam != "" {
		// конец дня
		tmp, err := time.Parse("2006-01-02", toParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to date format, use YYYY-MM-DD"})
			return
		}
		toTime = tmp.Add(24*time.Hour - time.Nanosecond)
	} else {
		toTime = time.Now()
	}

	// 1. Общее количество заказов
	var total int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM orders WHERE created_at >= $1 AND created_at <= $2`,
		fromTime, toTime,
	).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 2. Активные заказы (status != 'завершён')
	var active int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM orders WHERE status <> 'завершён' AND created_at >= $1 AND created_at <= $2`,
		fromTime, toTime,
	).Scan(&active)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 3. Выполненные заказы
	var completed int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM orders WHERE status = 'завершён' AND completed_at >= $1 AND completed_at <= $2`,
		fromTime, toTime,
	).Scan(&completed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 4. Среднее время выполнения (сек)
	var avgSecs sql.NullFloat64
	err = db.QueryRow(
		`SELECT EXTRACT(EPOCH FROM AVG(completed_at - created_at)) FROM orders WHERE status = 'завершён' AND completed_at >= $1 AND completed_at <= $2`,
		fromTime, toTime,
	).Scan(&avgSecs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := GeneralStatsResponse{
		TotalOrders:               total,
		ActiveOrders:              active,
		CompletedOrders:           completed,
		AverageCompletionTimeSecs: 0,
	}
	if avgSecs.Valid {
		resp.AverageCompletionTimeSecs = avgSecs.Float64
	}

	c.JSON(http.StatusOK, resp)
}
