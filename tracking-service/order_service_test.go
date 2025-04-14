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

func setupRouterTracking() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// Health-check endpoint.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// POST endpoint для обновления трекинга.
	r.POST("/tracking", func(c *gin.Context) {
		var t TrackingInfo
		if err := c.BindJSON(&t); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
			return
		}
		t.UpdatedAt = time.Now()
		// Эмулируем успех сохранения.
		c.JSON(http.StatusOK, t)
	})

	// GET endpoint для получения трекинга по order_id.
	r.GET("/tracking/:orderId", func(c *gin.Context) {
		// Для теста возвращаем заранее подготовленные данные.
		t := TrackingInfo{
			OrderID:   c.Param("orderId"),
			CourierID: "courier_123",
			Status:    "в пути",
			UpdatedAt: time.Now(),
		}
		c.JSON(http.StatusOK, t)
	})

	return r
}

func TestTrackingHealth(t *testing.T) {
	router := setupRouterTracking()
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Ожидался 200 OK, получено %d", w.Code)
	}
}

func TestPostTracking(t *testing.T) {
	router := setupRouterTracking()
	trackingPayload := TrackingInfo{
		OrderID:   "20250414010101",
		CourierID: "courier_123",
		Status:    "в пути",
	}
	payloadBytes, _ := json.Marshal(trackingPayload)
	req, _ := http.NewRequest("POST", "/tracking", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Ожидался статус 200, получено %d", w.Code)
	}
}

func TestGetTracking(t *testing.T) {
	router := setupRouterTracking()
	req, _ := http.NewRequest("GET", "/tracking/20250414010101", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Ожидался статус 200, получено %d", w.Code)
	}

	var tInfo TrackingInfo
	if err := json.Unmarshal(w.Body.Bytes(), &tInfo); err != nil {
		t.Fatalf("Ошибка разбора ответа: %v", err)
	}
	if tInfo.OrderID != "20250414010101" {
		t.Errorf("Ожидался order_id '20250414010101', получено %s", tInfo.OrderID)
	}
}
