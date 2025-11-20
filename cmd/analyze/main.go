// package main

// import (
// 	"context"
// 	"encoding/json"
// 	"example/my-go-api/app"
// 	"example/my-go-api/app/config"
// 	"example/my-go-api/app/models"
// 	"fmt"
// 	"log"
// 	"time"
// )

// func main() {
// 	// Load configuration
// 	cfg, err := config.LoadConfig()
// 	if err != nil {
// 		log.Fatalf("failed to load config: %v", err)
// 	}

// 	eng, err := app.NewUCIEngine(cfg.Engine.Path)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer eng.Close()
// 	_ = eng.NewGame()

// 	//load games from db
// 	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
// 	defer cancel()
// 	app.MustInitDB()
// 	games, err := app.LoadGames(ctx, cfg.User, 2) // last 3 games
// 	if err != nil {
// 		fmt.Printf("loadGames: %v", err)
// 	}
// 	if len(games) == 0 {
// 		fmt.Printf("no games found for %s", cfg.User)
// 		return
// 	}

// 	start := time.Now()
// 	for _, g := range games {
// 		fmt.Printf("\n=== Analyzing game: %s vs %s (%s) ===\n", cfg.User, g.Opponent, g.URL)

// 		denormalizedPgn := g.PGN

// 		tags := app.ParsePGNTags(denormalizedPgn)
// 		pgn := app.NormalizeChessDotComPGN(denormalizedPgn)

// 		meta := models.GameLite{
// 			URL: tags["Link"], When: app.GetUnixTimeStamp(tags["Date"], tags["StartTime"], tags["Timezone"]),
// 			Color: "white", Opponent: "demo", Result: tags["Result"],
// 			PGN: pgn, TimeControl: tags["TimeControl"],
// 		}

// 		report, err := app.AnalyzePGN(meta.PGN, meta, eng, cfg)
// 		if err != nil {
// 			panic(err)
// 		}

// 		b, _ := json.MarshalIndent(report, "", "  ")
// 		fmt.Println(string(b))
// 	}

// 	fmt.Printf("Took %s", time.Since(start))

// 	//pgn := `1. e4 e5 2. Qh5 Nc6 3. Bc4 Nf6 4. Qxf7# 1-0`
// 	//denormalizedPgn := "[Event \"Live Chess\"]\n[Site \"Chess.com\"]\n[Date \"2025.11.01\"]\n[Round \"-\"]\n[White \"waluyo912\"]\n[Black \"xpertwizard\"]\n[Result \"1-0\"]\n[CurrentPosition \"r2qr3/pb1n1kpB/2nbppN1/1p1p3Q/2pP1P2/P1P1P2R/1P1N2PP/R1B3K1 b - - 4 16\"]\n[Timezone \"UTC\"]\n[ECO \"D00\"]\n[ECOUrl \"https://www.chess.com/openings/Queens-Pawn-Opening-1...d5\"]\n[UTCDate \"2025.11.01\"]\n[UTCTime \"14:21:46\"]\n[WhiteElo \"1741\"]\n[BlackElo \"1734\"]\n[TimeControl \"600\"]\n[Termination \"waluyo912 won by resignation\"]\n[StartTime \"14:21:46\"]\n[EndDate \"2025.11.01\"]\n[EndTime \"14:25:15\"]\n[Link \"https://www.chess.com/game/live/144992844670\"]\n\n1. d4 {[%clk 0:09:43.8]} 1... d5 {[%clk 0:09:57.8]} 2. f4 {[%clk 0:09:42.4]} 2... Nf6 {[%clk 0:09:55.5]} 3. Nf3 {[%clk 0:09:38.3]} 3... c5 {[%clk 0:09:44.5]} 4. c3 {[%clk 0:09:36.9]} 4... e6 {[%clk 0:09:40]} 5. e3 {[%clk 0:09:36.1]} 5... Bd6 {[%clk 0:09:35.4]} 6. Bd3 {[%clk 0:09:35.4]} 6... c4 {[%clk 0:09:28]} 7. Bc2 {[%clk 0:09:34.1]} 7... O-O {[%clk 0:09:21.5]} 8. O-O {[%clk 0:09:32.8]} 8... Nc6 {[%clk 0:09:17.1]} 9. Nbd2 {[%clk 0:09:30.7]} 9... b5 {[%clk 0:08:50]} 10. a3 {[%clk 0:09:28.2]} 10... Re8 {[%clk 0:08:35.4]} 11. Ne5 {[%clk 0:09:25.6]} 11... Bb7 {[%clk 0:08:21]} 12. Rf3 {[%clk 0:09:23.6]} 12... Nd7 {[%clk 0:07:54.4]} 13. Rh3 {[%clk 0:09:19.8]} 13... f6 {[%clk 0:07:46.5]} 14. Bxh7+ {[%clk 0:09:15.7]} 14... Kf8 {[%clk 0:07:40.6]} 15. Ng6+ {[%clk 0:09:13.3]} 15... Kf7 {[%clk 0:07:36.6]} 16. Qh5 {[%clk 0:09:09.7]} 1-0\n"
// 	// denormalizedPgn := games[0].PGN

// 	// tags := app.ParsePGNTags(denormalizedPgn)
// 	// pgn := app.NormalizeChessDotComPGN(denormalizedPgn)

// 	// meta := models.GameLite{
// 	// 	URL: tags["Link"], When: app.GetUnixTimeStamp(tags["Date"], tags["StartTime"], tags["Timezone"]),
// 	// 	Color: "white", Opponent: "demo", Result: tags["Result"],
// 	// 	PGN: pgn, TimeControl: tags["TimeControl"],
// 	// }

// 	// // ~600ms per position so it’s quick
// 	// report, err := app.AnalyzePGN(meta.PGN, meta, eng)
// 	// if err != nil {
// 	// 	panic(err)
// 	// }

// 	// b, _ := json.MarshalIndent(report, "", "  ")
// 	// fmt.Println(string(b))

// 	// Bonus: direct engine ping (startpos) to confirm part 1 too
// 	// score, err := eng.EvalFEN(context.Background(),
// 	// 	"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", 400)
// 	// if err != nil {
// 	// 	panic(err)
// 	// }
// 	// fmt.Printf("\nstartpos bestmove=%s cp=%v mate=%v\n", score.Best, score.CP, score.Mate)
// }

package main

import (
	"context"
	"example/my-go-api/app"
	"example/my-go-api/app/config"
	"example/my-go-api/app/models"
	"fmt"
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
		fmt.Printf("no games found for %s\n", cfg.User)
		return
	}

	numWorkers := app.GetWorkerCount()
	fmt.Printf("Analyzing %d games with %d workers\n", len(games), numWorkers)

	jobs := make(chan models.GameLite, len(games))
	results := make(chan models.GameEval, len(games))
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

	var allResults []models.GameEval

	for res := range results {
		allResults = append(allResults, res)
	}

	fmt.Printf("Got %d successful results\n", len(allResults))

	fmt.Printf("Took %s\n", time.Since(start))
}
