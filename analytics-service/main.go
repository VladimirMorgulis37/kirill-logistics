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

var db *sql.DB

type CourierStat struct {
	CourierID             string  `json:"courier_id"`
	CourierName           string  `json:"courier_name"`
	CompletedOrders       int     `json:"completed_orders"`
	TotalRevenue          float64 `json:"total_revenue"`
	AverageDeliveryTimeSec float64 `json:"average_delivery_time_sec"`
}

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

func startRabbitConsumer() {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Ошибка подключения к RabbitMQ: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Ошибка открытия канала: %v", err)
	}

	_, err = ch.QueueDeclare(
		"order_created", true, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Ошибка объявления очереди: %v", err)
	}

	msgs, err := ch.Consume("order_created", "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Ошибка подписки на очередь: %v", err)
	}

	go func() {
		for msg := range msgs {
			log.Printf("Получено сообщение: %s", msg.Body)
			handleOrderCreated(msg.Body)
		}
	}()
}

func startCourierCreatedConsumer() {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Ошибка подключения к RabbitMQ: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Ошибка канала: %v", err)
	}

	_, err = ch.QueueDeclare("courier_created", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Ошибка объявления очереди: %v", err)
	}

	msgs, err := ch.Consume("courier_created", "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Ошибка подписки: %v", err)
	}

	go func() {
		for msg := range msgs {
			handleCourierCreated(msg.Body)
		}
	}()
}

func startOrderCompletedConsumer() {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Ошибка подключения к RabbitMQ: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Ошибка канала: %v", err)
	}

	_, err = ch.QueueDeclare("order_completed", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Ошибка объявления очереди: %v", err)
	}

	msgs, err := ch.Consume("order_completed", "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Ошибка consume: %v", err)
	}

	go func() {
		for msg := range msgs {
			log.Printf("Получено завершение заказа: %s", msg.Body)
			handleOrderCompleted(msg.Body)
		}
	}()
}

func startDeliveryCalculatedConsumer() {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Ошибка подключения к RabbitMQ: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Ошибка канала: %v", err)
	}

	_, err = ch.QueueDeclare("delivery_calculated", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Ошибка объявления очереди delivery_calculated: %v", err)
	}

	msgs, err := ch.Consume("delivery_calculated", "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Ошибка consume delivery_calculated: %v", err)
	}

	go func() {
		for msg := range msgs {
			log.Printf("Получено событие доставки: %s", msg.Body)
			HandleDeliveryCalculated(msg.Body)
		}
	}()
}

func handleOrderCreated(body []byte) {
	var evt map[string]string
	if err := json.Unmarshal(body, &evt); err != nil {
		log.Printf("Некорректный JSON в событии: %v", err)
		return
	}

	if evt["event"] == "order_created" {
		_, err := db.Exec(`UPDATE general_stats SET total_orders = total_orders + 1`)
		if err != nil {
			log.Printf("Ошибка увеличения total_orders: %v", err)
		} else {
			log.Println("Аналитика обновлена: общий заказ добавлен")
		}
	}

	if evt["status"] == "новый" {
		_, err := db.Exec(`UPDATE general_stats SET active_orders = active_orders + 1`)
		if err != nil {
			log.Printf("Ошибка увеличения active_orders: %v", err)
		} else {
			log.Println("Аналитика обновлена: активный заказ добавлен")
		}
	}
}

func handleCourierCreated(body []byte) {
	var evt map[string]string
	if err := json.Unmarshal(body, &evt); err != nil {
		log.Printf("Ошибка разбора courier_created: %v", err)
		return
	}

	if evt["event"] != "courier_created" {
		return
	}

	courierID := evt["courier_id"]
	courierName := evt["courier_name"]

	_, err := db.Exec(`
		INSERT INTO courier_stats (courier_id, courier_name, completed_orders, total_revenue, average_delivery_time_sec)
		VALUES ($1, $2, 0, 0, 0)
		ON CONFLICT (courier_id) DO NOTHING
	`, courierID, courierName)

	if err != nil {
		log.Printf("Ошибка вставки courier_stats: %v", err)
	} else {
		log.Printf("Добавлен курьер в статистику: %s (%s)", courierName, courierID)
	}
}


func handleOrderCompleted(body []byte) {
	var evt map[string]string
	if err := json.Unmarshal(body, &evt); err != nil {
		log.Printf("Некорректный JSON в order_completed: %v", err)
		return
	}

	if evt["event"] == "order_completed" && evt["status"] == "завершён" {
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Ошибка начала транзакции: %v", err)
			return
		}
		defer tx.Rollback()
		courierID := evt["courier_id"]
		if courierID == "" {
			log.Println("Нет courier_id в событии завершения заказа")
			return
		}

		createdAt, err1 := time.Parse(time.RFC3339, evt["created_at"])
		completedAt, err2 := time.Parse(time.RFC3339, evt["completed_at"])
		if err1 != nil || err2 != nil {
			log.Printf("Ошибка парсинга времени: %v | %v", err1, err2)
			return
		}
		duration := completedAt.Sub(createdAt).Seconds()
		// Получаем текущие значения
		var currentCount int
		var currentAvg float64
		err = db.QueryRow(`
			SELECT completed_orders, average_delivery_time_sec
			FROM courier_stats WHERE courier_id = $1
		`, courierID).Scan(&currentCount, &currentAvg)

		if err != nil && err != sql.ErrNoRows {
			log.Printf("Ошибка получения текущих данных courier_stats: %v", err)
			return
		}

		newCount := currentCount + 1
		newAvg := ((currentAvg * float64(currentCount)) + duration) / float64(newCount)

		// Вставка или обновление
		_, err = db.Exec(`
			INSERT INTO courier_stats (courier_id, completed_orders, average_delivery_time_sec, total_revenue)
			VALUES ($1, $2, $3, 0)
			ON CONFLICT (courier_id) DO UPDATE
			SET completed_orders = $2,
				average_delivery_time_sec = $3
		`, courierID, newCount, newAvg)

		if err != nil {
			log.Printf("Ошибка обновления courier_stats: %v", err)
		} else {
			log.Printf("Обновлена статистика курьера %s: заказов %d, среднее время %.2f сек", courierID, newCount, newAvg)
		}
		_, err = tx.Exec(`UPDATE general_stats SET completed_orders = completed_orders + 1`)
		if err != nil {
			log.Printf("Ошибка обновления completed_orders: %v", err)
			return
		}

		_, err = tx.Exec(`UPDATE general_stats SET active_orders = active_orders - 1 WHERE active_orders > 0`)
		if err != nil {
			log.Printf("Ошибка уменьшения active_orders: %v", err)
			return
		}

		if err := tx.Commit(); err != nil {
			log.Printf("Ошибка коммита: %v", err)
		} else {
			log.Println("Аналитика обновлена: заказ завершён")
		}
	}
}

func HandleDeliveryCalculated(body []byte) {
	var evt map[string]interface{}
	if err := json.Unmarshal(body, &evt); err != nil {
		log.Printf("Некорректный JSON в delivery_calculated: %v", err)
		return
	}

	if evt["event"] != "delivery_calculated" {
		return
	}

	courierID, ok1 := evt["courier_id"].(string)
	cost, ok2 := evt["cost"].(float64)

	if !ok1 || !ok2 || courierID == "" {
		log.Printf("Пропущены поля в delivery_calculated: courier_id или cost")
		return
	}

	_, err := db.Exec(`
		INSERT INTO courier_stats (courier_id, total_revenue, completed_orders, average_delivery_time_sec)
		VALUES ($1, $2, 0, 0)
		ON CONFLICT (courier_id) DO UPDATE
		SET total_revenue = courier_stats.total_revenue + $2
	`, courierID, cost)

	if err != nil {
		log.Printf("Ошибка обновления revenue в courier_stats: %v", err)
		return
	}

	log.Printf("Обновлена выручка курьера %s: +%.2f", courierID, cost)
}

func getCourierStats(c *gin.Context) {
	rows, err := db.Query(`
		SELECT courier_id, COALESCE(courier_name, ''), completed_orders, total_revenue, average_delivery_time_sec
		FROM courier_stats
		ORDER BY total_revenue DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []CourierStat
	for rows.Next() {
		var stat CourierStat
		if err := rows.Scan(&stat.CourierID, &stat.CourierName, &stat.CompletedOrders, &stat.TotalRevenue, &stat.AverageDeliveryTimeSec); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, stat)
	}
	c.JSON(http.StatusOK, list)
}

func main() {
	var err error
	db, err = initDB()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	go startRabbitConsumer()
	go startCourierCreatedConsumer()
	go startOrderCompletedConsumer()
	go startDeliveryCalculatedConsumer()


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

	// Health-check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Общая статистика заказов с фильтром по периоду
	r.GET("/analytics/general", getGeneralStats)
	r.GET("/analytics/couriers", getCourierStats)

	if err := r.Run(":8080	"); err != nil {
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
