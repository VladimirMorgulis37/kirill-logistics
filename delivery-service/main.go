package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
)

// DeliveryRequest содержит входные параметры для расчёта стоимости доставки.
type DeliveryRequest struct {
	FromLat  float64 `json:"from_lat"`  // Широта отправления
	FromLng  float64 `json:"from_lng"`  // Долгота отправления
	ToLat    float64 `json:"to_lat"`    // Широта доставки
	ToLng    float64 `json:"to_lng"`    // Долгота доставки
	Weight   float64 `json:"weight"`    // Вес посылки (кг)
	Length   float64 `json:"length"`    // Длина посылки (метры)
	Width    float64 `json:"width"`     // Ширина посылки (метры)
	Height   float64 `json:"height"`    // Высота посылки (метры)
	Urgency  int     `json:"urgency"`   // 1 — стандарт, 2 — экспресс
	OrderID   string  `json:"order_id"`   // 🆕 ID заказа
	CourierID string  `json:"courier_id"` // 🆕 ID курьера
}

// DeliveryResponse содержит рассчитанную стоимость доставки.
type DeliveryResponse struct {
	EstimatedCost float64 `json:"estimated_cost"`
	Currency      string  `json:"currency"`
}

var (
	baseFee       float64
	distanceRate  float64
	weightRate    float64
	volumeRate    float64
	urgencyFactor float64
	currency      string
)

// getEnv возвращает значение переменной окружения или defaultVal, если переменная не задана.
func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// initConfig инициализирует коэффициенты расчёта из переменных окружения.
func initConfig() {
	var err error
	baseFee, err = strconv.ParseFloat(getEnv("BASE_FEE", "50"), 64)
	if err != nil {
		baseFee = 50
	}
	distanceRate, err = strconv.ParseFloat(getEnv("DISTANCE_RATE", "5"), 64)
	if err != nil {
		distanceRate = 5
	}
	weightRate, err = strconv.ParseFloat(getEnv("WEIGHT_RATE", "2"), 64)
	if err != nil {
		weightRate = 2
	}
	volumeRate, err = strconv.ParseFloat(getEnv("VOLUME_RATE", "3"), 64)
	if err != nil {
		volumeRate = 3
	}
	urgencyFactor, err = strconv.ParseFloat(getEnv("URGENCY_FACTOR", "1.5"), 64)
	if err != nil {
		urgencyFactor = 1.5
	}
	currency = getEnv("CURRENCY", "USD")
}

// haversineDistance вычисляет расстояние в километрах между двумя координатами.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // радиус Земли в км
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// calculateVolume вычисляет объём посылки.
func calculateVolume(length, width, height float64) float64 {
	return length * width * height
}

// calculateDeliveryCost вычисляет стоимость доставки по заданной формуле.
func calculateDeliveryCost(req DeliveryRequest) float64 {
	distance := haversineDistance(req.FromLat, req.FromLng, req.ToLat, req.ToLng)
	volume := calculateVolume(req.Length, req.Width, req.Height)

	// Формула: стоимость = baseFee + (distance * distanceRate) + (weight * weightRate) + (volume * volumeRate)
	// Если заказ срочный (urgency > 1), стоимость умножается на urgencyFactor.
	cost := baseFee + (distance * distanceRate) + (req.Weight * weightRate) + (volume * volumeRate)
	if req.Urgency > 1 {
		cost *= urgencyFactor
	}
	return cost
}

func publishDeliveryCalculatedEvent(orderID, courierID string, cost float64) error {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	_, err = ch.QueueDeclare("delivery_calculated", true, false, false, false, nil)
	if err != nil {
		return err
	}

	event := map[string]interface{}{
		"event":      "delivery_calculated",
		"order_id":   orderID,
		"courier_id": courierID,
		"cost":       cost,
	}
	body, _ := json.Marshal(event)

	return ch.Publish("", "delivery_calculated", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

func main() {
	// Инициализируем конфигурацию
	initConfig()

	// Создаем роутер Gin
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})

	// POST /calculate — эндпоинт для расчета стоимости доставки.
	r.POST("/calculate", func(c *gin.Context) {
		var req DeliveryRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
			return
		}
		cost := calculateDeliveryCost(req)
		resp := DeliveryResponse{
			EstimatedCost: cost,
			Currency:      currency,
		}
		c.JSON(http.StatusOK, resp)
		if req.OrderID != "" {
			if err := publishDeliveryCalculatedEvent(req.OrderID, req.CourierID, cost); err != nil {
				log.Printf("Ошибка публикации delivery_calculated: %v", err)
			}
		}
	})

	r.Run(":8080")
}
