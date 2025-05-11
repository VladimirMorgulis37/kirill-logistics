package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

type CourierTracking struct {
	CourierID string    `json:"courier_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	UpdatedAt time.Time `json:"updated_at"`
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

func main() {
	// Инициализация БД.
	var err error
	db, err = initDB()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	// Инициализируем роутер Gin.
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
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})

	// WebSocket endpoint для получения обновлений местоположения.
	r.GET("/tracking/ws", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request)
	})

    r.POST("/couriers/tracking", func(c *gin.Context) {
        var ct CourierTracking
        if err := c.BindJSON(&ct); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
            return
        }
        ct.UpdatedAt = time.Now()
        // Сохраняем или обновляем запись
        _, err := db.Exec(`
            INSERT INTO courier_tracking
              (courier_id, latitude, longitude, updated_at)
            VALUES ($1,$2,$3, $4)
            ON CONFLICT (courier_id) DO UPDATE SET
			  latitude   = EXCLUDED.latitude,
			  longitude  = EXCLUDED.longitude,
			  updated_at = EXCLUDED.updated_at
        `, ct.CourierID, ct.Latitude, ct.Longitude, ct.UpdatedAt)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusCreated, ct)
    })
	r.GET("/couriers/tracking/:courierId", func(c *gin.Context) {
		courierId := c.Param("courierId")
		var ct CourierTracking
		query := "SELECT courier_id, latitude, longitude, updated_at FROM courier_tracking WHERE courier_id = $1"
		err := db.QueryRow(query, courierId).Scan(&ct.CourierID, &ct.Latitude, &ct.Longitude, &ct.UpdatedAt)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Курьер не найден"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, ct)
	})

	// Запускаем сервис на порту 8080.
	r.Run(":8080")
}
