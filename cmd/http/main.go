package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net/http"
	"time"
)

func main() {

	// create gin router
	r := gin.Default()

	// router for users
	r.GET("/bike", func(c *gin.Context) {
		user := c.Query("bike")
		if user == "" {
			fmt.Errorf("Модель велосипеда не указана")
			c.JSON(http.StatusOK, gin.H{"error": "Введите модель велосипеда !"})
			return
		}
	})

	log.Print("Сервер запущен на порту 8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Сбой в работе сервера: %v", err)
	}
}
