package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter(mockDB *sql.DB) *gin.Engine {
	db = mockDB
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	r.POST("/register", func(c *gin.Context) {
		var creds Credentials
		if err := c.BindJSON(&creds); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
			return
		}
		err := createUser(creds.Username, creds.Password, "admin")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"status": "Пользователь зарегистрирован"})
	})

	r.POST("/login", loginHandler)

	return r
}

func TestRegisterUser(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (username, password, role) VALUES ($1, $2, $3)")).
		WithArgs("testuser", "testpass", "admin").
		WillReturnResult(sqlmock.NewResult(1, 1))

	r := setupTestRouter(mockDB)
	body := Credentials{Username: "testuser", Password: "testpass"}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Пользователь зарегистрирован")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoginUser_Success(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, password, role FROM users WHERE username = $1")).
	WithArgs("testuser").
	WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password", "role"}).
			AddRow(1, "testuser", "testpass", "admin"))

	r := setupTestRouter(mockDB)
	body := Credentials{Username: "testuser", Password: "testpass"}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "token")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoginUser_WrongPassword(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, password, role FROM users WHERE username = $1")).
	WithArgs("testuser").
	WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password", "role"}).
			AddRow(1, "testuser", "correctpass", "admin"))

	r := setupTestRouter(mockDB)
	body := Credentials{Username: "testuser", Password: "wrongpass"}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Неверный логин или пароль")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoginUser_NotFound(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, password, role FROM users WHERE username = $1")).
	WithArgs("nonexistent").
	WillReturnError(sql.ErrNoRows)

	r := setupTestRouter(mockDB)
	body := Credentials{Username: "nonexistent", Password: "pass"}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Неверный логин или пароль")
	assert.NoError(t, mock.ExpectationsWereMet())
}
