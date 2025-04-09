package main

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt"
)

var jwtKey = []byte("your_secret_key")

// Credentials представляет структуру данных для входа.
type Credentials struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

func main() {
    r := gin.Default()

    // Простейший endpoint для проверки работоспособности сервиса.
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "ok"})
    })

    // Endpoint для авторизации: принимает логин и пароль, возвращает JWT.
    r.POST("/login", func(c *gin.Context) {
        var creds Credentials
        if err := c.BindJSON(&creds); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
            return
        }

        // Простейшая проверка – в реальной системе подключение к базе пользователей.
        if creds.Username != "admin" || creds.Password != "password" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            return
        }

        // Создание токена с временем жизни 1 час.
        expirationTime := time.Now().Add(1 * time.Hour)
        claims := &jwt.StandardClaims{
            ExpiresAt: expirationTime.Unix(),
            Subject:   creds.Username,
        }
        token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
        tokenString, err := token.SignedString(jwtKey)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create token"})
            return
        }

        c.JSON(http.StatusOK, gin.H{"token": tokenString})
    })

    // Пример защищённого маршрута.
    authorized := r.Group("/")
    authorized.Use(authMiddleware())
    {
        authorized.GET("/protected", func(c *gin.Context) {
            c.JSON(http.StatusOK, gin.H{"message": "This is a protected route."})
        })
    }

    // Слушаем порт 8080 (он будет проброшен через Docker)
    r.Run(":8080")
}

// authMiddleware проверяет наличие и валидность JWT.
func authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenString := c.GetHeader("Authorization")
        if tokenString == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
            return
        }
        claims := &jwt.StandardClaims{}
        token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
            return jwtKey, nil
        })
        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            return
        }
        c.Set("username", claims.Subject)
        c.Next()
    }
}
