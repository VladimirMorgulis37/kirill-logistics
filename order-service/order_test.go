package main

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type EventPublisher interface {
	PublishToQueue(queue string, payload any) error
}

// --- Мок EventPublisher ---
type MockPublisher struct {
	CalledQueue   string
	CalledPayload any
	WasCalled     bool
	FailPublish   bool
}

func (m *MockPublisher) PublishToQueue(queue string, payload any) error {
	m.CalledQueue = queue
	m.CalledPayload = payload
	m.WasCalled = true
	if m.FailPublish {
		return errors.New("ошибка публикации")
	}
	return nil
}

// --- Тесты публикации ---

func TestPublishNotification(t *testing.T) {
	mock := &MockPublisher{}
	err := publishNotificationWithPublisher(mock, "order_created", "user@example.com", "Ваш заказ создан")

	assert.NoError(t, err)
	assert.True(t, mock.WasCalled)
	assert.Equal(t, "notifications", mock.CalledQueue)

	payload, ok := mock.CalledPayload.(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, "order_created", payload["type"])
	assert.Equal(t, "user@example.com", payload["recipient"])
	assert.Equal(t, "Ваш заказ создан", payload["message"])
}

func TestPublishOrderCreatedEvent(t *testing.T) {
	mock := &MockPublisher{}
	err := publishOrderCreatedEventWithPublisher(mock, "order-123")

	assert.NoError(t, err)
	assert.True(t, mock.WasCalled)
	assert.Equal(t, "order_created", mock.CalledQueue)

	payload := mock.CalledPayload.(map[string]string)
	assert.Equal(t, "order-123", payload["order_id"])
	assert.Equal(t, "order_created", payload["event"])
	assert.Equal(t, "новый", payload["status"])
}

func TestPublishCourierCreatedEvent(t *testing.T) {
	mock := &MockPublisher{}
	err := publishCourierCreatedEventWithPublisher(mock, "courier-007", "Иван")

	assert.NoError(t, err)
	assert.True(t, mock.WasCalled)
	assert.Equal(t, "courier_created", mock.CalledQueue)

	payload := mock.CalledPayload.(map[string]string)
	assert.Equal(t, "courier-007", payload["courier_id"])
	assert.Equal(t, "Иван", payload["courier_name"])
}

func TestPublishOrderCompletedEvent(t *testing.T) {
	mock := &MockPublisher{}
	err := publishOrderCompletedEventWithPublisher(mock, "order-001", "courier-001", time.Now().Add(-5*time.Minute), time.Now())

	assert.NoError(t, err)
	assert.True(t, mock.WasCalled)
	assert.Equal(t, "order_completed", mock.CalledQueue)

	payload := mock.CalledPayload.(map[string]string)
	assert.Equal(t, "order-001", payload["order_id"])
	assert.Equal(t, "courier-001", payload["courier_id"])
	assert.Equal(t, "order_completed", payload["event"])
	assert.Equal(t, "завершён", payload["status"])
}

// --- Обёртки для мок-тестов ---

func publishNotificationWithPublisher(publisher EventPublisher, typ, recipient, message string) error {
	payload := map[string]string{
		"type":      typ,
		"recipient": recipient,
		"message":   message,
	}
	return publisher.PublishToQueue("notifications", payload)
}

func publishOrderCreatedEventWithPublisher(publisher EventPublisher, orderID string) error {
	return publisher.PublishToQueue("order_created", map[string]string{
		"event":    "order_created",
		"order_id": orderID,
		"status":   "новый",
	})
}

func publishCourierCreatedEventWithPublisher(publisher EventPublisher, courierID, courierName string) error {
	return publisher.PublishToQueue("courier_created", map[string]string{
		"event":        "courier_created",
		"courier_id":   courierID,
		"courier_name": courierName,
	})
}

func publishOrderCompletedEventWithPublisher(publisher EventPublisher, orderID, courierID string, createdAt, completedAt time.Time) error {
	return publisher.PublishToQueue("order_completed", map[string]string{
		"order_id":     orderID,
		"courier_id":   courierID,
		"created_at":   createdAt.Format(time.RFC3339),
		"completed_at": completedAt.Format(time.RFC3339),
		"event":        "order_completed",
		"status":       "завершён",
		"message":      "Заказ успешно завершён и доставлен",
	})
}
