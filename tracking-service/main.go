package main

import (
	"database/sql"
	"encoding/json"
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

// TrackingInfo описывает информацию трекинга с координатами.
type TrackingInfo struct {
	OrderID   string    `json:"order_id"`   // Идентификатор заказа
	CourierID string    `json:"courier_id"` // Идентификатор курьера
	Status    string    `json:"status"`     // Статус доставки, например "в пути"
	Latitude  float64   `json:"latitude"`   // GPS-широта
	Longitude float64   `json:"longitude"`  // GPS-долгота
	UpdatedAt time.Time `json:"updated_at"` // Время обновления
}

var db *sql.DB

// Конфигурация WebSocket
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
var wsClients = make(map[*websocket.Conn]bool)
var wsClientsMutex sync.Mutex

// initDB устанавливает соединение с базой данных Tracking Service.
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

// wsHandler обновляет соединение до WebSocket и добавляет клиента в пул.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Ошибка при апгрейде до WebSocket:", err)
		return
	}
	wsClientsMutex.Lock()
	wsClients[conn] = true
	wsClientsMutex.Unlock()
	log.Println("Новый клиент WebSocket подключен.")

	// Слушаем соединение, чтобы обрабатывать закрытие.
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			wsClientsMutex.Lock()
			delete(wsClients, conn)
			wsClientsMutex.Unlock()
			conn.Close()
			log.Println("WebSocket клиент отключился:", err)
			break
		}
	}
}

// broadcastTrackingUpdate рассылает обновленное местоположение курьера всем клиентам.
func broadcastTrackingUpdate(update TrackingInfo) {
	wsClientsMutex.Lock()
	defer wsClientsMutex.Unlock()
	message, err := json.Marshal(update)
	if err != nil {
		log.Println("Ошибка маршалинга обновления:", err)
		return
	}
	for client := range wsClients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Ошибка отправки WebSocket-сообщения: %v", err)
			client.Close()
			delete(wsClients, client)
		}
	}
	log.Println("Обновление местоположения отправлено:", update)
}

func main() {
	// Инициализация БД.
	var err error
	db, err = initDB()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	// Инициализируем роутер Gin.
	r := gin.Default()

	// Health-check endpoint.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})

	// WebSocket endpoint для получения обновлений местоположения.
	r.GET("/tracking/ws", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request)
	})

	// POST /tracking – принимает обновления от курьера.
	r.POST("/tracking", func(c *gin.Context) {
		var update TrackingInfo
		if err := c.BindJSON(&update); err != nil {
			log.Printf("Ошибка разбора JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
			return
		}
		update.UpdatedAt = time.Now()

		// Сохраняем обновление в БД (если требуется, можно добавить сохранение).
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
		if _, err := db.Exec(query, update.OrderID, update.CourierID, update.Status, update.Latitude, update.Longitude, update.UpdatedAt); err != nil {
			log.Printf("Ошибка обновления БД: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Рассылаем обновление всем подключенным клиентам.
		broadcastTrackingUpdate(update)

		c.JSON(http.StatusOK, update)
	})

	// GET /tracking/:orderId – возвращает актуальную информацию о местоположении для конкретного заказа.
	r.GET("/tracking/:orderId", func(c *gin.Context) {
		orderId := c.Param("orderId")
		var update TrackingInfo
		query := "SELECT order_id, courier_id, status, latitude, longitude, updated_at FROM tracking_info WHERE order_id = $1"
		err := db.QueryRow(query, orderId).Scan(&update.OrderID, &update.CourierID, &update.Status, &update.Latitude, &update.Longitude, &update.UpdatedAt)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Информация трекинга не найдена"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, update)
	})

	// Запускаем сервис на порту 8080.
	r.Run(":8080")
}
