package main

import (
    "net/http"
    "sync"
    "time"

    "github.com/gin-gonic/gin"
)

type Order struct {
    ID            string    `json:"id"`
    SenderName    string    `json:"sender_name"`
    RecipientName string    `json:"recipient_name"`
    AddressFrom   string    `json:"address_from"`
    AddressTo     string    `json:"address_to"`
    Status        string    `json:"status"`
    CreatedAt     time.Time `json:"created_at"`
}

var orders = make(map[string]Order)
var mu sync.Mutex

func main() {
    r := gin.Default()

    // Проверка работоспособности.
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "ok"})
    })

    // Создание заказа.
    r.POST("/orders", func(c *gin.Context) {
        var newOrder Order
        if err := c.BindJSON(&newOrder); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        newOrder.ID = generateID() // Генерация уникального идентификатора.
        newOrder.CreatedAt = time.Now()
        newOrder.Status = "новый"
        mu.Lock()
        orders[newOrder.ID] = newOrder
        mu.Unlock()
        c.JSON(http.StatusCreated, newOrder)
    })

    // Получение данных заказа по ID.
    r.GET("/orders/:id", func(c *gin.Context) {
        id := c.Param("id")
        mu.Lock()
        order, exists := orders[id]
        mu.Unlock()
        if !exists {
            c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
            return
        }
        c.JSON(http.StatusOK, order)
    })

    // Получение списка всех заказов.
    r.GET("/orders", func(c *gin.Context) {
        mu.Lock()
        var orderList []Order
        for _, o := range orders {
            orderList = append(orderList, o)
        }
        mu.Unlock()
        c.JSON(http.StatusOK, orderList)
    })

    // Обновление заказа.
    r.PUT("/orders/:id", func(c *gin.Context) {
        id := c.Param("id")
        mu.Lock()
        order, exists := orders[id]
        mu.Unlock()
        if !exists {
            c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
            return
        }
        if err := c.BindJSON(&order); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        orders[id] = order
        c.JSON(http.StatusOK, order)
    })

    // Удаление заказа.
    r.DELETE("/orders/:id", func(c *gin.Context) {
        id := c.Param("id")
        mu.Lock()
        _, exists := orders[id]
        if exists {
            delete(orders, id)
        }
        mu.Unlock()
        if !exists {
            c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
            return
        }
        c.JSON(http.StatusOK, gin.H{"status": "deleted"})
    })

    r.Run(":8080")
}

// generateID создает простой уникальный идентификатор (не для продакшена).
func generateID() string {
    return time.Now().Format("20060102150405")
}
