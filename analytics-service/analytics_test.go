package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHandleOrderCreated(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()
	db = mockDB

	event := map[string]string{"event": "order_created", "status": "новый"}
	body, _ := json.Marshal(event)

	mock.ExpectExec("UPDATE general_stats SET total_orders = total_orders \\+ 1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE general_stats SET active_orders = active_orders \\+ 1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	handleOrderCreated(body)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleCourierCreated(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()
	db = mockDB

	event := map[string]string{"event": "courier_created", "courier_id": "c1", "courier_name": "Ivan"}
	body, _ := json.Marshal(event)

	mock.ExpectExec("INSERT INTO courier_stats .*").
		WithArgs("c1", "Ivan").
		WillReturnResult(sqlmock.NewResult(1, 1))

	handleCourierCreated(body)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleOrderCompleted(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()
	db = mockDB

	createdAt := time.Now().Add(-10 * time.Minute).Format(time.RFC3339)
	completedAt := time.Now().Format(time.RFC3339)
	event := map[string]string{
		"event":       "order_completed",
		"status":      "завершён",
		"courier_id":  "c1",
		"created_at":  createdAt,
		"completed_at": completedAt,
	}
	body, _ := json.Marshal(event)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT completed_orders, average_delivery_time_sec").
		WithArgs("c1").
		WillReturnRows(sqlmock.NewRows([]string{"completed_orders", "average_delivery_time_sec"}).
			AddRow(2, 300.0))
	mock.ExpectExec("INSERT INTO courier_stats .*").
		WithArgs("c1", 3, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE general_stats SET completed_orders = completed_orders \\+ 1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE general_stats SET active_orders = active_orders - 1 .*").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	handleOrderCompleted(body)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleDeliveryCalculated(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()
	db = mockDB

	event := map[string]interface{}{"event": "delivery_calculated", "courier_id": "c1", "cost": 45.5}
	body, _ := json.Marshal(event)

	mock.ExpectExec("INSERT INTO courier_stats .*").
		WithArgs("c1", 45.5).
		WillReturnResult(sqlmock.NewResult(0, 1))

	HandleDeliveryCalculated(body)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCourierStats(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()
	db = mockDB

	rows := sqlmock.NewRows([]string{"courier_id", "courier_name", "completed_orders", "total_revenue", "average_delivery_time_sec"}).
		AddRow("c1", "Ivan", 5, 100.0, 300.0)

	mock.ExpectQuery("SELECT courier_id.*FROM courier_stats").
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest("GET", "/analytics/couriers", nil)
	c.Request = req

	getCourierStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Ivan")
}

func TestGetGeneralStats(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()
	db = mockDB

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM orders WHERE created_at >= .*").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM orders WHERE status <> 'завершён' .*").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM orders WHERE status = 'завершён' .*").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(6))
	mock.ExpectQuery("SELECT EXTRACT.*AVG\\(completed_at - created_at\\).*").
		WillReturnRows(sqlmock.NewRows([]string{"avg"}).AddRow(120.5))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest("GET", "/analytics/general?from=2023-01-01&to=2023-12-31", nil)
	c.Request = req

	getGeneralStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "total_orders")
}
