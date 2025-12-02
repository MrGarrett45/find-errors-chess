package main

import (
	"example/my-go-api/app"

	"github.com/gin-gonic/gin"
)

func main() {
	app.MustInitDB()
	router := gin.Default()
	router.GET("chessgames/:username", app.GetChessGames)
	router.GET("errors/:username", app.GetErrorPositions)
	router.Run("0.0.0.0:8080")
}
