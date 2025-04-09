package main

import (
    "net/http"
    "sync"
    "time"

    "github.com/gin-gonic/gin"
)

type TrackingInfo struct {
    OrderID   string    `json:"order_id"`
    CourierID string    `json:"courier_id"`
    Status    string    `json:"status"`
    UpdatedAt time.Time `json:"updated_at"`
}

var trackingData = make(map[string]TrackingInfo)
var mu sync.Mutex

func main() {
    r := gin.Default()

    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "ok"})
    })

    // Обновление статуса доставки.
    r.POST("/tracking", func(c *gin.Context) {
        var info TrackingInfo
        if err := c.BindJSON(&info); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        info.UpdatedAt = time.Now()
        mu.Lock()
        trackingData[info.OrderID] = info
        mu.Unlock()
        c.JSON(http.StatusOK, info)
    })

    // Получение статуса доставки по OrderID.
    r.GET("/tracking/:orderId", func(c *gin.Context) {
        orderId := c.Param("orderId")
        mu.Lock()
        info, exists := trackingData[orderId]
        mu.Unlock()
        if !exists {
            c.JSON(http.StatusNotFound, gin.H{"error": "Tracking info not found"})
            return
        }
        c.JSON(http.StatusOK, info)
    })

    r.Run(":8080")
}
