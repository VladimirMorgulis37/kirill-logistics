package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func main() {
	// Идентификатор заказа (уже созданный заказ в Order Service)
	orderID := "20250415225701" // Например, заказ, созданный ранее

	// Эмуляция процесса доставки: курьер "начинает" доставку, обновляются GPS-координаты и т.д.
	log.Println("Курьер приступает к доставке заказа", orderID)
	// Здесь можно эмулировать обновление местоположения, если требуется.

	// Эмулируем время доставки.
	time.Sleep(10 * time.Second)

	// Курьер завершает доставку, нажимая "Завершить заказ"
	url := "http://localhost:8082/orders/" + orderID + "/finish"
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		log.Fatalf("Ошибка создания запроса: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatalf("Ошибка декодирования ответа: %v", err)
	}
	log.Printf("Ответ сервера: %+v", result)
}
