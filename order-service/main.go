package main

import (
	"bytes"
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

type Courier struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Phone        string  `json:"phone"`
	VehicleType  string  `json:"vehicle_type"`  // "foot", "bike", "car"
	Status       string  `json:"status"`        // "available", "busy", "offline"
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	ActiveOrder  string  `json:"active_order_id"` // может быть "" если не назначен
}

// Order описывает заказ с расширенными полями для физический характеристик.
type Order struct {
	ID            string    `json:"id"`
	SenderName    string    `json:"sender_name"`
	RecipientName string    `json:"recipient_name"`
	AddressFrom   string    `json:"address_from"`
	AddressTo     string    `json:"address_to"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	CompletedAt   sql.NullTime `json:"completed_at"`
	// Новые поля для расчёта доставки
	Weight  float64 `json:"weight"`   // вес посылки (кг)
	Length  float64 `json:"length"`   // длина (метры)
	Width   float64 `json:"width"`    // ширина (метры)
	Height  float64 `json:"height"`   // высота (метры)
	Urgency int     `json:"urgency"`  // 1 - стандартная, 2 - экспресс

	CourierID string `json:"courier_id"`
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

func publishEventToQueue(queueName string, payload any) error {
	rabbitURL := os.Getenv("RABBITMQ_URL") // пример: "amqp://guest:guest@rabbitmq:5672/"
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return fmt.Errorf("ошибка подключения к RabbitMQ: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("ошибка открытия канала: %w", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(
		queueName, true, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("ошибка объявления очереди: %w", err)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %w", err)
	}

	return ch.Publish(
		"", queueName, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
}

func publishCourierCreatedEvent(courierID, courierName string) error {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("channel: %w", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare("courier_created", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("queue declare: %w", err)
	}

	event := map[string]string{
		"event":        "courier_created",
		"courier_id":   courierID,
		"courier_name": courierName,
	}
	body, _ := json.Marshal(event)

	return ch.Publish("", "courier_created", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

// publishOrderCreatedEvent публикует событие создания заказа
func publishOrderCreatedEvent(orderID string) error {
	rabbitURL := os.Getenv("RABBITMQ_URL") // пример: "amqp://guest:guest@rabbitmq:5672/"
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return fmt.Errorf("ошибка подключения к RabbitMQ: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("ошибка открытия канала: %w", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(
		"order_created", // Название очереди
		true,            // durable
		false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("ошибка объявления очереди: %w", err)
	}

	body, err := json.Marshal(map[string]string{
		"event":    "order_created",
		"order_id": orderID,
		"status":   "новый",
	})
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %w", err)
	}

	return ch.Publish(
		"", "order_created", false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
}

// Получение списка курьеров
func getCouriersHandler(c *gin.Context) {
    rows, err := db.Query("SELECT id, name FROM couriers")
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()

    var list []Courier
    for rows.Next() {
        var cur Courier
        if err := rows.Scan(&cur.ID, &cur.Name); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        list = append(list, cur)
    }
    c.JSON(http.StatusOK, list)
}

// Создание нового курьера
func createCourierHandler(c *gin.Context) {
    var cur Courier
    if err := c.BindJSON(&cur); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
        return
    }
    cur.ID = time.Now().Format("20060102150405")
	_, err := db.Exec(`
		INSERT INTO couriers (id, name, phone, vehicle_type, status, latitude, longitude, active_order_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, cur.ID, cur.Name, cur.Phone, cur.VehicleType, cur.Status, cur.Latitude, cur.Longitude, cur.ActiveOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 🔔 Публикация события courier_created
	if err := publishCourierCreatedEvent(cur.ID, cur.Name); err != nil {
		log.Printf("Ошибка отправки события courier_created: %v", err)
	}
    // 2) Отправляем POST в tracking-service на localhost
	endpoint := "http://tracking-service:8080/couriers/tracking"
    rec := map[string]interface{}{
        "courier_id": cur.ID,
        "latitude":   cur.Latitude,
        "longitude":  cur.Longitude,
    }
    payload, _ := json.Marshal(rec)
    resp, err := http.Post(endpoint, "application/json", bytes.NewReader(payload))
    if err != nil {
        log.Printf("createCourierHandler: POST to %s failed: %v", endpoint, err)
    } else {
        log.Printf("createCourierHandler: %s -> %s", endpoint, resp.Status)
        resp.Body.Close()
    }

    c.JSON(http.StatusCreated, cur)
}

func assignCourierHandler(c *gin.Context) {
    orderID := c.Param("id")

    // 1. Парсим входной JSON
    var body struct {
        CourierID string `json:"courier_id"`
    }
    if err := c.BindJSON(&body); err != nil {
        log.Printf("assignCourierHandler: bind JSON error: %v", err)
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 2. Начинаем транзакцию (чтобы обновления orders и couriers были атомарными)
    tx, err := db.Begin()
    if err != nil {
        log.Printf("assignCourierHandler: begin tx error: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка начала транзакции"})
        return
    }
    defer tx.Rollback()

    // 3. Обновляем orders и couriers в зависимости от body.CourierID
    var res sql.Result
    if body.CourierID == "" {
        // отвязываем курьера
        res, err = tx.Exec("UPDATE orders SET courier_id = NULL WHERE id = $1", orderID)
        if err == nil {
            // убираем active_order_id у всех курьеров, у которых он был
            _, err = tx.Exec("UPDATE couriers SET active_order_id = NULL WHERE active_order_id = $1", orderID)
        }
    } else {
        // привязываем курьера
        res, err = tx.Exec("UPDATE orders SET courier_id = $1 WHERE id = $2", body.CourierID, orderID)
        if err == nil {
            _, err = tx.Exec("UPDATE couriers SET active_order_id = $1 WHERE id = $2", orderID, body.CourierID)
        }
    }
    if err != nil {
        log.Printf("assignCourierHandler: DB update error: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления заказа или курьера"})
        return
    }

    // 4. Проверяем, был ли обновлён хоть один заказ
    rowsAffected, err := res.RowsAffected()
    if err != nil {
        log.Printf("assignCourierHandler: RowsAffected error: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка проверки обновления заказа"})
        return
    }
    if rowsAffected == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "Заказ не найден"})
        return
    }

    // 5. Коммитим транзакцию
    if err := tx.Commit(); err != nil {
        log.Printf("assignCourierHandler: commit tx error: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения изменений"})
        return
    }
	var lat, lon float64
	err = db.QueryRow(`SELECT latitude, longitude FROM couriers WHERE id = $1`, body.CourierID).Scan(&lat, &lon)
	if err != nil {
		log.Printf("assignCourierHandler: не удалось получить координаты курьера %s: %v", body.CourierID, err)
		lat, lon = 0, 0 // fallback
	}
    // 6. Вызываем tracking-service (если задан TRACKING_URL)
    if trackingURL := os.Getenv("TRACKING_URL"); trackingURL != "" && body.CourierID != "" {
        rec := map[string]interface{}{
            "order_id":   orderID,
            "courier_id": body.CourierID,
			"latitude":   lat,
    		"longitude":  lon,
        }
        payload, _ := json.Marshal(rec)
        endpoint := fmt.Sprintf("%s/couriers/tracking", trackingURL)
        log.Printf("assignCourierHandler: POST %s payload=%s", endpoint, payload)
        resp, err := http.Post(endpoint, "application/json", bytes.NewReader(payload))
        if err != nil {
            log.Printf("assignCourierHandler: POST to tracking failed: %v", err)
            // не возвращаем ошибку пользователю, т.к. основной кейс уже выполнен
        } else {
            resp.Body.Close()
        }
    }

    // 7. Отправляем единый JSON-ответ
    c.JSON(http.StatusOK, gin.H{
        "status":     "Курьер обновлён",
        "courier_id": body.CourierID,
    })
}

// publishOrderCompletedEvent публикует событие завершённого заказа в RabbitMQ.
func publishOrderCompletedEvent(orderID string, courierID string, createdAt, completedAt time.Time) error {
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
		"courier_id": courierID,
		"created_at":   createdAt.Format(time.RFC3339),
		"completed_at": completedAt.Format(time.RFC3339),
		"event":    "order_completed",
		"status":   "завершён",
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
		// 🔔 Публикация события order_created
		if err := publishOrderCreatedEvent(o.ID); err != nil {
			log.Printf("Ошибка публикации события order_created: %v", err)
		}
		c.JSON(http.StatusCreated, o)
	})

	r.GET("/orders", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, sender_name, recipient_name, address_from, address_to, status, created_at, completed_at, weight, length, width, height, urgency, COALESCE(courier_id, '') FROM orders")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		var orders []Order
		for rows.Next() {
			var o Order
			if err := rows.Scan(&o.ID, &o.SenderName, &o.RecipientName, &o.AddressFrom, &o.AddressTo, &o.Status, &o.CreatedAt, &o.CompletedAt, &o.Weight, &o.Length, &o.Width, &o.Height, &o.Urgency, &o.CourierID); err != nil {
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
		query := "SELECT id, sender_name, recipient_name, address_from, address_to, status, created_at, completed_at, weight, length, width, height, urgency, COALESCE(courier_id, '') FROM orders WHERE id = $1"
		err := db.QueryRow(query, id).Scan(&o.ID, &o.SenderName, &o.RecipientName, &o.AddressFrom, &o.AddressTo, &o.Status, &o.CreatedAt,&o.CompletedAt, &o.Weight, &o.Length, &o.Width, &o.Height, &o.Urgency, &o.CourierID)
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
		query := "UPDATE orders SET status = $1, completed_at = NOW() WHERE id = $2"
		_, err := db.Exec(query, "завершён", orderID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// 2. Получаем courier_id из заказа
		var courierID string
		err = db.QueryRow("SELECT courier_id FROM orders WHERE id = $1", orderID).Scan(&courierID)
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения курьера: " + err.Error()})
			return
		}
		if courierID != "" {
			// 3. Обновляем статус курьера на "доступен" и убираем active_order_id
			_, err = db.Exec(`UPDATE couriers SET status = 'доступен', active_order_id = NULL WHERE id = $1`, courierID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления статуса курьера: " + err.Error()})
				return
			}
		}

		var createdAt time.Time
		var completedAt sql.NullTime

		err = db.QueryRow(`SELECT created_at, completed_at FROM orders WHERE id = $1`, orderID).Scan(&createdAt, &completedAt)
		if err != nil {
			log.Printf("Ошибка получения дат заказа %s: %v", orderID, err)
			// можно не прерывать — просто не публиковать
			return
		}

		// Проверка на completedAt
		if !completedAt.Valid {
			log.Printf("completed_at пустой для заказа %s", orderID)
			return
		}
		// Публикуем событие в RabbitMQ, чтобы уведомить Notification Service.
		if err := publishOrderCompletedEvent(orderID, courierID, createdAt, completedAt.Time); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка отправки уведомления: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "Заказ завершён и уведомление отправлено"})
	})

	// (Дополнительно можно добавить GET /orders/:id и другие CRUD-эндпоинты)
	// Endpoint для удаления заказа
	r.DELETE("/orders/:id", func(c *gin.Context) {
		id := c.Param("id")
		result, err := db.Exec("DELETE FROM orders WHERE id = $1", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Заказ не найден"})
			return
		}
		// Успешно удалили — возвращаем 204 No Content
		c.Status(http.StatusNoContent)
	})

	    // 2. Эндпоинты для работы с курьерами
		r.GET("/couriers", getCouriersHandler)
		r.POST("/couriers", createCourierHandler)

		// 3. Привязка курьера к заказу
		r.PUT("/orders/:id/assign-courier", assignCourierHandler)

	r.Run(":8080")
}
