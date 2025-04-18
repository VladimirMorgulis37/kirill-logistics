package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Order описывает структуру заказа, возвращаемую Order Service.
type Order struct {
	ID            string  `json:"id"`
	SenderName    string  `json:"sender_name"`
	RecipientName string  `json:"recipient_name"`
	AddressFrom   string  `json:"address_from"`
	AddressTo     string  `json:"address_to"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"created_at"`
	Weight        float64 `json:"weight"`
	Length        float64 `json:"length"`
	Width         float64 `json:"width"`
	Height        float64 `json:"height"`
	Urgency       int     `json:"urgency"`
}

// DeliveryRequest содержит параметры для расчёта доставки.
type DeliveryRequest struct {
	FromLat float64 `json:"from_lat"`
	FromLng float64 `json:"from_lng"`
	ToLat   float64 `json:"to_lat"`
	ToLng   float64 `json:"to_lng"`
	Weight  float64 `json:"weight"`
	Length  float64 `json:"length"`
	Width   float64 `json:"width"`
	Height  float64 `json:"height"`
	Urgency int     `json:"urgency"`
}

// DeliveryResponse получает рассчитанную стоимость доставки.
type DeliveryResponse struct {
	EstimatedCost float64 `json:"estimated_cost"`
	Currency      string  `json:"currency"`
}

// TrackingInfo — данные, которые отправляются в Tracking Service.
type TrackingInfo struct {
	OrderID   string  `json:"order_id"`
	CourierID string  `json:"courier_id"`
	Status    string  `json:"status"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func main() {
	// 0) Настройки
	orderID := "20250418205340"          // замените на ваш реальный orderID
	trackingURL := "http://localhost:8083/tracking"
	courierID := "courier_123"

	// 1) Получаем данные заказа
	orderURL := "http://localhost:8082/orders/" + orderID
	resp, err := http.Get(orderURL)
	if err != nil {
		log.Fatalf("Ошибка получения информации о заказе: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("При получении заказа получен статус %d", resp.StatusCode)
	}

	var order Order
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		log.Fatalf("Ошибка декодирования ответа Order Service: %v", err)
	}
	log.Printf("Получен заказ: %+v", order)

	// 2) Расчёт стоимости доставки
	deliveryReq := DeliveryRequest{
		FromLat: order.AddressFromLat(), // если есть функция для получения координат
		FromLng: order.AddressFromLng(),
		ToLat:   order.AddressToLat(),
		ToLng:   order.AddressToLng(),
		Weight:  order.Weight,
		Length:  order.Length,
		Width:   order.Width,
		Height:  order.Height,
		Urgency: order.Urgency,
	}
	payload, err := json.Marshal(deliveryReq)
	if err != nil {
		log.Fatalf("Ошибка маршалинга запроса доставки: %v", err)
	}
	deliveryURL := "http://localhost:8086/calculate"
	dRes, err := http.Post(deliveryURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Fatalf("Ошибка запроса расчёта доставки: %v", err)
	}
	defer dRes.Body.Close()
	var deliveryResp DeliveryResponse
	if err := json.NewDecoder(dRes.Body).Decode(&deliveryResp); err != nil {
		log.Fatalf("Ошибка декодирования ответа Delivery Cost Service: %v", err)
	}
	log.Printf("Расчитанная стоимость доставки: %.2f %s", deliveryResp.EstimatedCost, deliveryResp.Currency)

	// 3) Эмуляция процесса доставки + отправка трекинга каждую 2 секунды
	log.Printf("Курьер приступает к доставке заказа %s...", orderID)
	lat := deliveryReq.FromLat
	lon := deliveryReq.FromLng
	for i := 0; i < 5; i++ {
		update := TrackingInfo{
			OrderID:   orderID,
			CourierID: courierID,
			Status:    "в пути",
			Latitude:  lat,
			Longitude: lon,
		}
		data, _ := json.Marshal(update)
		resp, err := http.Post(trackingURL, "application/json", bytes.NewReader(data))
		if err != nil {
			log.Printf("Ошибка отправки трекинга: %v", err)
		} else {
			log.Printf("Трекинг отправлен: %s", resp.Status)
			resp.Body.Close()
		}
		// Сдвигаем координаты для эмуляции движения
		lat += 0.01
		lon += 0.01
		time.Sleep(2 * time.Second)
	}

	// 4) Завершаем заказ в Order Service
	finishURL := "http://localhost:8082/orders/" + orderID + "/finish"
	reqFinish, err := http.NewRequest("PUT", finishURL, nil)
	if err != nil {
		log.Fatalf("Ошибка создания запроса для завершения заказа: %v", err)
	}
	client := &http.Client{}
	respFinish, err := client.Do(reqFinish)
	if err != nil {
		log.Fatalf("Ошибка выполнения запроса для завершения заказа: %v", err)
	}
	defer respFinish.Body.Close()
	var finishResult map[string]interface{}
	if err := json.NewDecoder(respFinish.Body).Decode(&finishResult); err != nil {
		log.Fatalf("Ошибка декодирования ответа Order Service: %v", err)
	}
	log.Printf("Заказ завершён. Ответ Order Service: %+v", finishResult)
}

// Ниже—пример заглушек, если адреса в заказе хранятся в виде строк.
// Замените на вашу реальную логику получения координат.
func (o Order) AddressFromLat() float64 { return 55.7558 }
func (o Order) AddressFromLng() float64 { return 37.6176 }
func (o Order) AddressToLat() float64   { return 59.9311 }
func (o Order) AddressToLng() float64   { return 30.3609 }
