package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// TrackingInfo описывает структуру, соответствующую данным, которые принимает Tracking Service.
type TrackingInfo struct {
	OrderID   string    `json:"order_id"`   // Идентификатор заказа
	CourierID string    `json:"courier_id"` // Идентификатор курьера
	Status    string    `json:"status"`     // Текущий статус (например, "в пути")
	Latitude  float64   `json:"latitude"`   // GPS-широта
	Longitude float64   `json:"longitude"`  // GPS-долгота
	UpdatedAt time.Time `json:"updated_at"` // Время обновления
}

func main() {
	// Адрес Tracking Service (убедитесь, что порт совпадает с настройками docker-compose)
	trackingURL := "http://localhost:8083/tracking"

	// Задаем идентификаторы заказа и курьера
	orderID := "20250414010101"
	courierID := "courier_123"
	status := "в пути"

	// Начальные координаты (пример: Москва)
	lat, long := 55.7558, 37.6176

	// Периодическое отправление обновлений
	for {
		// Собираем данные о текущем местоположении
		update := TrackingInfo{
			OrderID:   orderID,
			CourierID: courierID,
			Status:    status,
			Latitude:  lat,
			Longitude: long,
			UpdatedAt: time.Now(),
		}

		// Преобразуем данные в JSON
		body, err := json.Marshal(update)
		if err != nil {
			log.Printf("Ошибка маршалинга JSON: %v", err)
			continue
		}

		// Отправляем POST-запрос Tracking Service
		resp, err := http.Post(trackingURL, "application/json", bytes.NewReader(body))
		if err != nil {
			log.Printf("Ошибка отправки POST запроса: %v", err)
		} else {
			log.Printf("Обновление успешно отправлено, HTTP статус: %s", resp.Status)
			resp.Body.Close()
		}

		// Эмуляция движения: изменяем координаты (в данном примере добавляем небольшое значение)
		lat += 0.0001
		long += 0.0001

		// Задержка перед следующим обновлением (например, 5 секунд)
		time.Sleep(5 * time.Second)
	}
}
