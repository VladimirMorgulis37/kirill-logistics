package main

import (
    "net/http"
    "sync"

    "github.com/gin-gonic/gin"
)

// Для примера используем in-memory хранилище заказов (в реальности данные нужно брать из БД или Order Service).
type Order struct {
    Status string `json:"status"`
}

var orders = []Order{
    {Status: "новый"},
    {Status: "завершён"},
    {Status: "новый"},
}

var mu sync.Mutex

func main() {
    r := gin.Default()

    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "ok"})
    })

    // Endpoint для получения простой статистики.
    r.GET("/analytics/stats", func(c *gin.Context) {
        mu.Lock()
        newCount, completedCount := 0, 0
        for _, order := range orders {
            if order.Status == "новый" {
                newCount++
            } else if order.Status == "завершён" {
                completedCount++
            }
        }
        mu.Unlock()

        stats := gin.H{
            "total_orders": len(orders),
            "new_orders":   newCount,
            "completed_orders": completedCount,
        }
        c.JSON(http.StatusOK, stats)
    })

    r.Run(":8080")
}
