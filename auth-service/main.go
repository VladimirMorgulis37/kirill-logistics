package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	_ "github.com/lib/pq"
)

// jwtKey – секретный ключ для подписывания токенов (в продакшене лучше получать его из переменных окружения)
var jwtKey = []byte("your_secret_key")

// User представляет пользователя, как он хранится в базе данных.
type User struct {
	ID       int
	Username string
	Password string // В production системе следует хранить хэш пароля!
	Role     string
}

// Credentials – структура для входа/регистрации.
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Глобальное подключение к базе данных.
var db *sql.DB

// initDB устанавливает соединение с базой данных, используя переменные окружения.
// Требуется, чтобы в переменных окружения были заданы: DB_HOST, DB_USER, DB_PASSWORD, DB_NAME.
func initDB() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := "5432" // PostgreSQL в контейнере слушает на 5432

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	database, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}
	if err = database.Ping(); err != nil {
		return nil, err
	}
	return database, nil
}

func main() {
	var err error
	db, err = initDB()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	r := gin.Default()
	corsConfig := cors.Config{
        AllowOrigins:     []string{"http://localhost:3000"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }
    r.Use(cors.New(corsConfig))
	// Health-check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Endpoint для регистрации нового пользователя.
	r.POST("/register", func(c *gin.Context) {
		var creds Credentials
		if err := c.BindJSON(&creds); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
			return
		}
		// По умолчанию регистрируем пользователя с ролью "admin". Логику можно расширить по необходимости.
		err := createUser(creds.Username, creds.Password, "admin")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"status": "Пользователь зарегистрирован"})
	})

	// Endpoint для логина – проверяет данные из БД, сравнивает пароль и генерирует JWT.
	r.POST("/login", loginHandler)

	// Пример защищённого маршрута, для которого действует middleware проверки токена.
	authorized := r.Group("/")
	authorized.Use(authMiddleware())
	{
		authorized.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "Доступ защищённого ресурса"})
		})
	}

	r.Run(":8080")
}

// loginHandler обрабатывает запрос на логин.
func loginHandler(c *gin.Context) {
	var creds Credentials
	if err := c.BindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
		return
	}
	user, err := getUserByUsername(creds.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный логин или пароль"})
		return
	}
	// Для примера сравнение происходит в открытом виде. В production используется хэширование паролей.
	if user.Password != creds.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный логин или пароль"})
		return
	}
	// Генерация JWT-токена с временем жизни 1 час.
	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &jwt.StandardClaims{
		ExpiresAt: expirationTime.Unix(),
		Subject:   user.Username,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании токена"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// authMiddleware проверяет наличие и корректность JWT-токена в заголовке Authorization.
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Отсутствует токен"})
			return
		}
		claims := &jwt.StandardClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Некорректный токен"})
			return
		}
		c.Set("username", claims.Subject)
		c.Next()
	}
}

// getUserByUsername выполняет запрос к таблице users для поиска пользователя по username.
func getUserByUsername(username string) (User, error) {
	var user User
	query := "SELECT id, username, password, role FROM users WHERE username = $1"
	err := db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Password, &user.Role)
	return user, err
}

// createUser вставляет нового пользователя в таблицу users.
func createUser(username, password, role string) error {
	query := "INSERT INTO users (username, password, role) VALUES ($1, $2, $3)"
	_, err := db.Exec(query, username, password, role)
	return err
}
