package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Order описывает структуру заказа из Order Service.
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
	CourierID     string  `json:"courier_id"`
}

// GeoResult хранит ответ геокодера.
type GeoResult struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

// DeliveryRequest содержит данные для расчёта стоимости.
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
	OrderID   string  `json:"order_id"`   // 🆕 ID заказа
	CourierID string  `json:"courier_id"` // 🆕 ID курьера
}

// DeliveryResponse хранит ответ сервиса расчёта стоимости.
type DeliveryResponse struct {
	EstimatedCost float64 `json:"estimated_cost"`
	Currency      string  `json:"currency"`
}

// TrackingInfo отправляется в Tracking Service.
type CourierTracking struct {
	CourierID string  `json:"courier_id"`
	Status    string  `json:"status"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	UpdatedAt string  `json:"updated_at"`
}

// moveTowards плавно перемещает курьера к целевой точке, отправляя трекинг.
func moveTowards(lat, lon *float64, targetLat, targetLon float64, steps int, orderID, courierID, trackingURL string) {
	stepLat := (targetLat - *lat) / float64(steps)
	stepLon := (targetLon - *lon) / float64(steps)

	for i := 0; i < steps; i++ {
		*lat += stepLat
		*lon += stepLon

		update := CourierTracking{
			CourierID: courierID,
			Latitude:  *lat,
			Longitude: *lon,
			UpdatedAt: time.Now().Format(time.RFC3339),
		}
		data, err := json.Marshal(update)
		if err != nil {
			log.Printf("Ошибка маршалинга JSON: %v", err)
			continue
		}
		resp, err := http.Post(trackingURL, "application/json", bytes.NewReader(data))
		if err != nil {
			log.Printf("Ошибка трекинга: %v", err)
		} else {
			log.Printf("📍 Трекинг %d: lat=%.5f, lon=%.5f", i+1, *lat, *lon)
			resp.Body.Close()
		}
		sleepDuration := time.Duration(1000+rand.Intn(4000)) * time.Millisecond
		log.Printf("Задержка на %v", sleepDuration)
		time.Sleep(sleepDuration)
	}
}

// geocode получает координаты по текстовому адресу через Nominatim.
func geocode(address string) (float64, float64, error) {
	endpoint := "https://nominatim.openstreetmap.org/search?format=json&q=" + url.QueryEscape(address)
	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("User-Agent", "courier-simulator/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("геокодер: %w", err)
	}
	defer resp.Body.Close()

	var results []GeoResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return 0, 0, fmt.Errorf("распаковка геоданных: %w", err)
	}
	if len(results) == 0 {
		return 0, 0, fmt.Errorf("адрес не найден: %s", address)
	}

	lat, _ := strconv.ParseFloat(results[0].Lat, 64)
	lon, _ := strconv.ParseFloat(results[0].Lon, 64)
	return lat, lon, nil
}

func main() {
	// Параметры эмуляции (подставьте ваши)
	orderID := "20250514130551"
	orderURL := "http://localhost:8082/orders/" + orderID
	trackingURL := "http://localhost:8083/couriers/tracking"
	deliveryURL := "http://localhost:8086/calculate"

	// 1) Получаем заказ
	resp, err := http.Get(orderURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Fatalf("не удалось получить заказ %s: %v", orderID, err)
	}
	defer resp.Body.Close()
	var order Order
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		log.Fatalf("ошибка декодирования заказа: %v", err)
	}
	log.Printf("Заказ: %+v", order)

	courierID := order.CourierID
	if courierID == "" {
		log.Fatalf("❌ У заказа %s не назначен courier_id", orderID)
	}

	// 2) Геокодинг отправителя и получателя
	fromLat, fromLng, err := geocode(order.AddressFrom)
	if err != nil {
		log.Fatalf("геокодирование отправителя: %v", err)
	}
	toLat, toLng, err := geocode(order.AddressTo)
	if err != nil {
		log.Fatalf("геокодирование получателя: %v", err)
	}
	log.Printf("Отправитель: %.5f, %.5f; Получатель: %.5f, %.5f", fromLat, fromLng, toLat, toLng)

	// 3) Расчёт стоимости доставки
	dReq := DeliveryRequest{
		FromLat: fromLat, FromLng: fromLng,
		ToLat: toLat, ToLng: toLng,
		Weight: order.Weight, Length: order.Length,
		Width: order.Width, Height: order.Height,
		Urgency: order.Urgency,
		OrderID:  order.ID,          // 🟢 добавлено
		CourierID: order.CourierID,  // 🟢 добавлено
	}
	b, _ := json.Marshal(dReq)
	dResp, err := http.Post(deliveryURL, "application/json", bytes.NewReader(b))
	if err != nil {
		log.Fatalf("расчёт доставки: %v", err)
	}
	defer dResp.Body.Close()
	var dRes DeliveryResponse
	if err := json.NewDecoder(dResp.Body).Decode(&dRes); err != nil {
		log.Fatalf("декодирование стоимости: %v", err)
	}
	log.Printf("Стоимость: %.2f %s", dRes.EstimatedCost, dRes.Currency)

	// 4) Текущие координаты курьера
	ctURL := "http://localhost:8083/couriers/tracking/" + courierID
	respCT, err := http.Get(ctURL)
	if err != nil || respCT.StatusCode != http.StatusOK {
		log.Fatalf("координаты курьера: %v", err)
	}
	defer respCT.Body.Close()
	var ct CourierTracking
	if err := json.NewDecoder(respCT.Body).Decode(&ct); err != nil {
		log.Fatalf("декодирование трекинга: %v", err)
	}
	lat := ct.Latitude
	lon := ct.Longitude
	log.Printf("Старт: %.5f, %.5f", lat, lon)

	// 5) Эмуляция маршрута: к отправителю, затем к получателю
	moveTowards(&lat, &lon, fromLat, fromLng, 10, orderID, courierID, trackingURL)
	moveTowards(&lat, &lon, toLat, toLng, 15, orderID, courierID, trackingURL)

	// 6) Завершение заказа
	finishURL := orderURL + "/finish"
	rf, _ := http.NewRequest("PUT", finishURL, nil)
	client := &http.Client{}
	resF, err := client.Do(rf)
	if err != nil || resF.StatusCode != http.StatusOK {
		log.Fatalf("завершение заказа: %v", err)
	}
	defer resF.Body.Close()
	log.Println("Заказ завершён")
}
