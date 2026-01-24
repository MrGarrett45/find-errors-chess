package main

import (
	"example/my-go-api/app"
	"log"
)

func main() {
	app.MustInitDB()
	app.InitStripe()
	router, err := app.NewRouter()
	if err != nil {
		log.Fatalf("failed to initialize router: %v", err)
	}
	router.Run("0.0.0.0:8080")
}
