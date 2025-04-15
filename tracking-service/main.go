package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

// TrackingInfo описывает информацию трекинга, включая GPS-координаты.
type TrackingInfo struct {
	OrderID   string    `json:"order_id"`   // Идентификатор заказа
	CourierID string    `json:"courier_id"` // Идентификатор курьера
	Status    string    `json:"status"`     // Статус доставки ("в пути", "доставлен" и т.д.)
	Latitude  float64   `json:"latitude"`   // GPS-широта
	Longitude float64   `json:"longitude"`  // GPS-долгота
	UpdatedAt time.Time `json:"updated_at"` // Время обновления
}

var db *sql.DB

// WebSocket-конфигурация.
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
var wsClients = make(map[*websocket.Conn]bool)
var wsClientsMutex sync.Mutex

// initDB устанавливает соединение с базой данных, параметры берутся из переменных окружения.
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

	// WebSocket endpoint для real-time обновлений.
	r.GET("/tracking/ws", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request)
	})

	// POST /tracking: обновление (или вставка) данных о трекинге с GPS-координатами.
	r.POST("/tracking", func(c *gin.Context) {
		var t TrackingInfo
		if err := c.BindJSON(&t); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
			return
		}
		t.UpdatedAt = time.Now()
		query := `
			INSERT INTO tracking_info (order_id, courier_id, status, latitude, longitude, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (order_id) DO UPDATE SET
				courier_id = EXCLUDED.courier_id,
				status = EXCLUDED.status,
				latitude = EXCLUDED.latitude,
				longitude = EXCLUDED.longitude,
				updated_at = EXCLUDED.updated_at;
		`
		_, err := db.Exec(query, t.OrderID, t.CourierID, t.Status, t.Latitude, t.Longitude, t.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Отправляем обновление всем подключенным клиентам WebSocket.
		broadcastTrackingUpdate(t)
		c.JSON(http.StatusOK, t)
	})

	// GET /tracking/:orderId: получение данных трекинга для конкретного заказа.
	r.GET("/tracking/:orderId", func(c *gin.Context) {
		orderId := c.Param("orderId")
		var t TrackingInfo
		query := "SELECT order_id, courier_id, status, latitude, longitude, updated_at FROM tracking_info WHERE order_id = $1"
		err := db.QueryRow(query, orderId).Scan(&t.OrderID, &t.CourierID, &t.Status, &t.Latitude, &t.Longitude, &t.UpdatedAt)
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

	// Запускаем сервис на порту 8080.
	r.Run(":8080")
}

// wsHandler обновляет соединение до WebSocket и добавляет клиента в список.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Ошибка при апгрейде до WebSocket:", err)
		return
	}
	wsClientsMutex.Lock()
	wsClients[conn] = true
	wsClientsMutex.Unlock()

	// Слушаем соединение, чтобы отреагировать на закрытие.
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			wsClientsMutex.Lock()
			delete(wsClients, conn)
			wsClientsMutex.Unlock()
			conn.Close()
			break
		}
	}
}

// broadcastTrackingUpdate рассылает обновление трекинга всем подключенным WebSocket клиентам.
func broadcastTrackingUpdate(t TrackingInfo) {
	wsClientsMutex.Lock()
	defer wsClientsMutex.Unlock()
	for client := range wsClients {
		err := client.WriteJSON(t)
		if err != nil {
			log.Printf("Ошибка отправки WebSocket-сообщения: %v", err)
			client.Close()
			delete(wsClients, client)
		}
	}
}
