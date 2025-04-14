package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// setupRouterAnalytics эмулирует работу сервиса аналитики.
func setupRouterAnalytics() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// Health-check endpoint.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Эндпоинт аналитики.
	// Для тестирования возвращаем фиксированные данные.
	r.GET("/analytics/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"total_orders":     100,
			"new_orders":       20,
			"completed_orders": 80,
		})
	})
	return r
}

func TestAnalyticsHealth(t *testing.T) {
	router := setupRouterAnalytics()
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Ожидался 200 OK, получено %d", w.Code)
	}
}

func TestAnalyticsStats(t *testing.T) {
	router := setupRouterAnalytics()
	req, _ := http.NewRequest("GET", "/analytics/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Ожидался 200 OK, получено %d", w.Code)
	}

	var stats map[string]int
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatalf("Ошибка разбора ответа: %v", err)
	}
	if stats["total_orders"] != 100 {
		t.Errorf("Ожидалось total_orders = 100, получено %d", stats["total_orders"])
	}
}
