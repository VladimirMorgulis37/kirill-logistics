// auth_service_test.go
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

// Настроим тестовый роутер; для тестов можем вызвать только нужные эндпоинты.
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// Health-check endpoint.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Эндпоинт логина с минимальной логикой (заменим здесь обращение к DB фиктивным условием)
	r.POST("/login", func(c *gin.Context) {
		var creds Credentials
		if err := c.BindJSON(&creds); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
			return
		}
		// Для теста используем условие: если логин "admin" и пароль "password", всё ок.
		if creds.Username != "admin" || creds.Password != "password" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный логин или пароль"})
			return
		}

		expirationTime := time.Now().Add(1 * time.Hour)
		claims := &jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			Subject:   creds.Username,
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(jwtKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания токена"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": tokenString})
	})

	// Пример защищённого эндпоинта.
	authorized := r.Group("/")
	authorized.Use(authMiddleware())
	{
		authorized.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "Доступ защищённого ресурса"})
		})
	}

	return r
}

func TestHealthEndpoint(t *testing.T) {
	router := setupRouter()

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Ошибка разбора ответа: %v", err)
	}
	if response["status"] != "ok" {
		t.Errorf("Expected status ok, got %s", response["status"])
	}
}

func TestLoginAndProtected(t *testing.T) {
	router := setupRouter()

	// Тест логина.
	loginPayload := Credentials{
		Username: "admin",
		Password: "password",
	}
	body, _ := json.Marshal(loginPayload)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on login, got %d", w.Code)
	}

	var loginResp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("Ошибка разбора ответа логина: %v", err)
	}
	token, exists := loginResp["token"]
	if !exists || token == "" {
		t.Fatal("Токен не получен")
	}

	// Тест защищённого эндпоинта.
	req, _ = http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on protected endpoint, got %d", w.Code)
	}

	var protectedResp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &protectedResp); err != nil {
		t.Fatalf("Ошибка разбора ответа защищённого эндпоинта: %v", err)
	}
	if protectedResp["message"] != "Доступ защищённого ресурса" {
		t.Errorf("Unexpected message: %s", protectedResp["message"])
	}
}
