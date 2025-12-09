package main

import (
	"time"

	"example/my-go-api/app"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	app.MustInitDB()
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
		MaxAge:       12 * time.Hour,
	}))

	router.GET("chessgames/:username", app.GetChessGames)
	router.GET("errors/:username", app.GetErrorPositions)
	router.Run("0.0.0.0:8080")
}
