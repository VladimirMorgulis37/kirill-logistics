package main

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Заглушка для email-функции
var sendEmailMockCalled bool
func fakeSendEmail(to, subject, body string) error {
	sendEmailMockCalled = true
	if to == "fail@example.com" {
		return errors.New("email failed")
	}
	return nil
}

func TestProcessNotification_Success(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()
	db = mockDB

	// Переопределим функцию отправки письма
	sendEmailFunc = fakeSendEmail
	sendEmailMockCalled = false // сброс флага

	msg := NotificationMessage{
		Type:      "order_completed",
		Recipient: "success@example.com",
		Message:   "Ваш заказ доставлен",
	}

	// Мокаем SQL
	mock.ExpectQuery("INSERT INTO notifications .* RETURNING id").
		WithArgs(msg.Type, msg.Recipient, msg.Message, "pending", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec("UPDATE notifications SET status = .*").
		WithArgs("sent", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Тестируем
	err := processNotification(msg)
	assert.NoError(t, err)
	assert.True(t, sendEmailMockCalled)
	assert.NoError(t, mock.ExpectationsWereMet())
}


func TestProcessNotification_EmailFailure(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()
	db = mockDB
	sendEmailFunc = fakeSendEmail

	msg := NotificationMessage{
		Type:      "order_completed",
		Recipient: "fail@example.com",
		Message:   "Ошибка почты",
	}

	mock.ExpectQuery("INSERT INTO notifications .* RETURNING id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
	mock.ExpectExec("UPDATE notifications SET status = .*").
		WithArgs("failed", 2).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := processNotification(msg)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func setupRouterWithMockDB(mockDB *sql.DB) *gin.Engine {
	db = mockDB
	gin.SetMode(gin.TestMode)

	r := gin.New() // вместо gin.Default()
	r.Use(gin.Recovery()) // добавляем минимальный middleware

	r.GET("/notifications", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, type, recipient, message, status, created_at FROM notifications ORDER BY created_at DESC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		var notifications []Notification
		for rows.Next() {
			var n Notification
			if err := rows.Scan(&n.ID, &n.Type, &n.Recipient, &n.Message, &n.Status, &n.CreatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			notifications = append(notifications, n)
		}
		c.JSON(http.StatusOK, notifications)
	})

	r.GET("/notifications/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}
		var n Notification
		query := "SELECT id, type, recipient, message, status, created_at FROM notifications WHERE id = $1"
		err = db.QueryRow(query, id).Scan(&n.ID, &n.Type, &n.Recipient, &n.Message, &n.Status, &n.CreatedAt)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Уведомление не найдено"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(http.StatusOK, n)
	})

	return r
}

func TestGetAllNotifications(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()

	createdAt := time.Now()
	mock.ExpectQuery("SELECT id, type, recipient, message, status, created_at FROM notifications.*").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "recipient", "message", "status", "created_at"}).
			AddRow(1, "order_completed", "test@example.com", "Заказ доставлен", "sent", createdAt))

	r := setupRouterWithMockDB(mockDB)
	req := httptest.NewRequest("GET", "/notifications", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test@example.com")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNotificationByID_Found(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()

	createdAt := time.Now()
	mock.ExpectQuery("SELECT id, type, recipient, message, status, created_at FROM notifications WHERE id = \\$1").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "type", "recipient", "message", "status", "created_at",
		}).AddRow(1, "order_completed", "one@example.com", "Доставлено", "sent", createdAt))

	r := setupRouterWithMockDB(mockDB)
	req := httptest.NewRequest("GET", "/notifications/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "one@example.com")
}

func TestGetNotificationByID_NotFound(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()

	mock.ExpectQuery("SELECT id, type, recipient, message, status, created_at FROM notifications WHERE id =").
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	r := setupRouterWithMockDB(mockDB)
	req := httptest.NewRequest("GET", "/notifications/999", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "не найдено")
}
