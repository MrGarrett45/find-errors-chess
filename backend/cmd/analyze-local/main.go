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
	settings := models.EngineSettings{Depth: 12, MoveTimeMS: 75, UseDepth: false}

	app.MustInitDB()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	app.ProcessBatch(ctx, cfg, models.JobMessage{
		User:           "xpertwizard",
		BatchIndex:     0,
		NumGames:       100,
		EngineDepth:    settings.Depth,
		EngineMoveTime: settings.MoveTimeMS,
		EngineUseDepth: settings.UseDepth,
	})
	log.Printf("Took %s", time.Since(start))
}
