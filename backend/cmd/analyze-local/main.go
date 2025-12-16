package main

import (
	"context"
	"example/my-go-api/app"
	"example/my-go-api/app/config"
	"example/my-go-api/app/models"
	"log"
	"time"
)

func main() {
	start := time.Now()
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	app.MustInitDB()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	app.ProcessBatch(ctx, cfg, models.JobMessage{User: "xpertwizard", BatchIndex: 0, NumGames: 500})
	log.Printf("Took %s", time.Since(start))
}
