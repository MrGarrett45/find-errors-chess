package main

import (
	"context"
	"example/my-go-api/app"
	"example/my-go-api/app/config"
	"example/my-go-api/app/models"
	"log"
	"sync"
	"time"
)

func main() {
	start := time.Now()
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	app.MustInitDB()
	//load last x number of games
	games, err := app.LoadGames(ctx, cfg.User, cfg.Engine.NumGames)
	if err != nil {
		log.Fatalf("loadGames: %v", err)
	}
	if len(games) == 0 {
		log.Printf("no games found for %s", cfg.User)
		return
	}

	numWorkers := app.GetWorkerCount()
	log.Printf("Analyzing %d games with %d workers", len(games), numWorkers)

	jobs := make(chan models.GameLite, len(games))
	results := make(chan models.GameLite, len(games))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			eng, err := app.NewUCIEngine(cfg.Engine.Path)
			if err != nil {
				log.Printf("worker %d: failed to create engine: %v", id, err)
				return
			}
			defer eng.Close()
			_ = eng.NewGame()

			for g := range jobs {
				if report, err := app.AnalyzeOneGame(cfg, eng, g); err != nil {
					log.Printf("worker %d: error analyzing game %s: %v", id, g.URL, err)
				} else {
					results <- report
				}
			}
		}(i)
	}

	// Feed jobs
	go func() {
		defer close(jobs)
		for _, g := range games {
			jobs <- g
		}
	}()

	// Close results once ALL workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	var allResults []models.GameLite

	for res := range results {
		allResults = append(allResults, res)
	}

	ctx2, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	err = app.SaveMoves(ctx2, cfg, allResults)
	if err != nil {
		log.Fatalf("failed to save moves: %v", err)
	}

	log.Printf("Got %d successful results", len(allResults))
	log.Printf("Took %s", time.Since(start))
}
