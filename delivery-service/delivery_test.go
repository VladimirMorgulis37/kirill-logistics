package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHaversineDistance(t *testing.T) {
	// Москва - Санкт-Петербург ≈ 633 км
	dist := haversineDistance(55.7558, 37.6173, 59.9343, 30.3351)
	assert.InDelta(t, 633, dist, 5)
}

func TestCalculateVolume(t *testing.T) {
	volume := calculateVolume(1.0, 0.5, 0.2)
	assert.InDelta(t, 0.1, volume, 0.0001)
}

func TestCalculateDeliveryCost(t *testing.T) {
	initConfig()

	req := DeliveryRequest{
		FromLat:  55.7558,
		FromLng:  37.6173,
		ToLat:    59.9343,
		ToLng:    30.3351,
		Weight:   2.0,
		Length:   1.0,
		Width:    0.5,
		Height:   0.2,
		Urgency:  1, // стандарт
	}

	cost := calculateDeliveryCost(req)
	assert.True(t, cost > 0)
}

func TestCalculateDeliveryCostWithUrgency(t *testing.T) {
	initConfig()

	req := DeliveryRequest{
		FromLat:  55.7558,
		FromLng:  37.6173,
		ToLat:    59.9343,
		ToLng:    30.3351,
		Weight:   2.0,
		Length:   1.0,
		Width:    0.5,
		Height:   0.2,
		Urgency:  2, // экспресс
	}

	standard := calculateDeliveryCost(DeliveryRequest{FromLat: req.FromLat, FromLng: req.FromLng, ToLat: req.ToLat, ToLng: req.ToLng, Weight: req.Weight, Length: req.Length, Width: req.Width, Height: req.Height, Urgency: 1})
	urgent := calculateDeliveryCost(req)

	assert.True(t, urgent > standard)
	assert.InDelta(t, urgent, standard*urgencyFactor, 0.01)
}

func TestCalculateHandler(t *testing.T) {
	// Подготовка
	gin.SetMode(gin.TestMode)
	initConfig()

	router := gin.Default()
	router.POST("/calculate", func(c *gin.Context) {
		var req DeliveryRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
			return
		}
		cost := calculateDeliveryCost(req)
		c.JSON(http.StatusOK, DeliveryResponse{
			EstimatedCost: cost,
			Currency:      currency,
		})
	})

	req := DeliveryRequest{
		FromLat:  55.75,
		FromLng:  37.62,
		ToLat:    59.93,
		ToLng:    30.33,
		Weight:   2.0,
		Length:   1.0,
		Width:    0.5,
		Height:   0.5,
		Urgency:  1,
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/calculate", bytes.NewBuffer(body))
	r.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp DeliveryResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.True(t, resp.EstimatedCost > 0)
	assert.Equal(t, currency, resp.Currency)
}
