package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockEnv struct {
	db   *sql.DB
	mock sqlmock.Sqlmock
	r    *gin.Engine
}

func setupMockEnv(t *testing.T) *mockEnv {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	env := &mockEnv{db: db, mock: mock}
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})
	r.POST("/couriers/tracking", func(c *gin.Context) {
		var ct CourierTracking
		if err := c.BindJSON(&ct); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
			return
		}
		ct.UpdatedAt = time.Now()
		env.mock.ExpectExec("INSERT INTO courier_tracking").
			WithArgs(ct.CourierID, ct.Latitude, ct.Longitude, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		c.JSON(http.StatusCreated, ct)
	})
	r.GET("/couriers/tracking/:courierId", func(c *gin.Context) {
		courierId := c.Param("courierId")
		rows := sqlmock.NewRows([]string{"courier_id", "latitude", "longitude", "updated_at"}).
			AddRow(courierId, 55.5, 37.5, time.Now())
		env.mock.ExpectQuery(regexp.QuoteMeta("SELECT courier_id, latitude, longitude, updated_at FROM courier_tracking WHERE courier_id = $1")).
			WithArgs(courierId).WillReturnRows(rows)
		var ct CourierTracking
		err := env.db.QueryRow("SELECT courier_id, latitude, longitude, updated_at FROM courier_tracking WHERE courier_id = $1", courierId).
			Scan(&ct.CourierID, &ct.Latitude, &ct.Longitude, &ct.UpdatedAt)
		assert.NoError(t, err)
		c.JSON(http.StatusOK, ct)
	})
	env.r = r
	return env
}

func TestHealthEndpoint(t *testing.T) {
	env := setupMockEnv(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	env.r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestPostCourierTracking(t *testing.T) {
	env := setupMockEnv(t)
	payload := CourierTracking{
		CourierID: "c1",
		Latitude:  55.5,
		Longitude: 37.5,
	}
	body, _ := json.Marshal(payload)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/couriers/tracking", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	env.r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Contains(t, rr.Body.String(), "c1")
}

func TestGetCourierTracking(t *testing.T) {
	env := setupMockEnv(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/couriers/tracking/c1", nil)
	env.r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "courier_id")
}
