package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// Order уже определён в основном коде order-service.
// Для тестов мы будем эмулировать работу без подключения к реальной БД.

func setupRouterOrder() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// Health-check endpoint.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Endpoint для создания заказа.
	// Здесь вместо реальной записи в БД возвращаем заказ, как будто он сохранён.
	r.POST("/orders", func(c *gin.Context) {
		var order Order
		if err := c.BindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
			return
		}
		order.ID = time.Now().Format("20060102150405")
		order.CreatedAt = time.Now()
		order.Status = "новый"
		c.JSON(http.StatusCreated, order)
	})
	return r
}

func TestOrderHealth(t *testing.T) {
	router := setupRouterOrder()
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Ожидался 200 OK, получено %d", w.Code)
	}
}

func TestCreateOrder(t *testing.T) {
	router := setupRouterOrder()
	orderPayload := Order{
		SenderName:    "Иван",
		RecipientName: "Петр",
		AddressFrom:   "ул. Ленина 1",
		AddressTo:     "ул. Гагарина 10",
		Status:        "новый",
	}
	payloadBytes, _ := json.Marshal(orderPayload)
	req, _ := http.NewRequest("POST", "/orders", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Ожидался статус 201, получено %d", w.Code)
	}

	var createdOrder Order
	if err := json.Unmarshal(w.Body.Bytes(), &createdOrder); err != nil {
		t.Fatalf("Ошибка разбора ответа: %v", err)
	}

	if createdOrder.ID == "" {
		t.Error("Ожидался сгенерированный ID, получен пустой ID")
	}
}
