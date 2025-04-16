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
// Здесь координаты задаются вручную, остальные параметры берутся из заказа.
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

func main() {
	// Укажите идентификатор заказа, который был создан в Order Service.
	orderID := "20250416165108" // замените на реальный orderID

	// Шаг 1. Получаем информацию о заказе из Order Service (без координат)
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

	// Шаг 2. Формируем запрос в Delivery Cost Service, используя параметры из заказа.
	deliveryReq := DeliveryRequest{
		// Задаем координаты вручную (или можно дополнять, если они сохранены в заказе)
		FromLat: 55.7558,  // например, координаты отправления (Москва)
		FromLng: 37.6176,
		ToLat:   59.9311,  // например, координаты доставки (Санкт-Петербург)
		ToLng:   30.3609,
		// Берем параметры из заказа
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

	// Эмуляция процесса доставки
	log.Printf("Курьер приступает к доставке заказа %s...", orderID)
	time.Sleep(10 * time.Second) // эмулируем время доставки

	// Шаг 3. Завершаем заказ. Курьер посылает запрос на завершение заказа в Order Service.
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
