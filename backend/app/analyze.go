// --- analyze.go ---
package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"example/my-go-api/app/config"
	"example/my-go-api/app/models"

	"github.com/notnil/chess"
)

const (
	InaccuracyThreshold        = 50  // 0.50 pawns
	MistakeThreshold           = 100 // 1.00 pawns
	BlunderThreshold           = 200 // 2.00 pawns
	OpeningInaccuracyThreshold = 30
)

func AnalyzePGN(meta models.GameLite, eng *UCIEngine, cfg *config.Config, username string) ([]models.Move, error) {
	// Parse PGN into new game
	g := chess.NewGame()
	if err := g.UnmarshalText([]byte(meta.PGN)); err != nil {
		return []models.Move{}, err
	}
	positions := g.Positions()

	// Collect FEN snapshots of each position
	var fens []models.FENEval
	for i, p := range positions {
		if i >= cfg.Engine.NumMoves {
			break
		}
		fenEval := fenInfoFromPosition(p)

		fens = append(fens, fenEval)

	}

	// New game (lets the engine clear its internal state)
	eng.NewGame()

	// Evaluate each picked FEN with ~500ms each (tweakable)
	ctx := context.Background()
	for i := range fens {
		c2, cancel := context.WithTimeout(ctx, 2*time.Second)
		score, _ := eng.EvalFEN(c2, fens[i].FEN, cfg)
		cancel()
		fens[i].Score = score
	}

	var moves []models.Move
	for i, m := range g.Moves() {
		if i >= cfg.Engine.NumMoves {
			break
		}

		color := "w"
		if !IsEven(i) {
			color = "b"
		}

		playedBy := username
		if string(meta.Color[0]) != color {
			playedBy = meta.Opponent
		}

		moveNumber := (i / 2) + 1

		//make sure fenAfter exists for any given position before using it
		var fenAfter models.FENEval
		if i+1 < len(fens) {
			fenAfter = fens[i+1]
		} else {
			fenAfter = models.FENEval{}
		}

		uciStr := chess.UCINotation{}.Encode(nil, m)
		sanStr := ""
		if i < len(positions) {
			sanStr = chess.AlgebraicNotation{}.Encode(positions[i], m)
		}

		moveAnalysis := GetMoveAnalysis(color, fens[i], fenAfter)
		moves = append(moves, models.Move{
			MoveUCI:    uciStr,
			MoveSAN:    sanStr,
			PlayedBy:   playedBy,
			FenBefore:  fens[i],
			FenAfter:   fenAfter,
			MoveNumber: moveNumber,
			Ply:        i + 1,
			Color:      color,
			Analysis:   moveAnalysis,
		})
	}

	return moves, nil
}

func fenInfoFromPosition(pos *chess.Position) models.FENEval {
	fen := pos.String()

	//this is inverted for some reason
	side := "w"
	if pos.Turn() == chess.Black {
		side = "b"
	}
	// Full move number is at the end of FEN; chess.Position doesn’t expose it directly,
	// but notnil/chess puts it in pos.String(). We'll parse minimally:
	// We know pos.String() is a FEN, so we'll split and read items.
	parts := strings.Split(string(fen), " ")
	moveNum := 1
	if len(parts) >= 6 {
		fmt.Sscanf(parts[5], "%d", &moveNum)
	}
	return models.FENEval{
		MoveNumber: moveNum,
		SideToMove: side,
		FEN:        string(fen),
	}
}

// What we let our workers call to process games
func AnalyzeOneGame(cfg *config.Config, eng *UCIEngine, g models.GameLite, username string) (models.GameLite, error) {
	log.Printf("Analyzing game: %s vs %s (%s)", username, g.Opponent, g.URL)

	g.PGN = NormalizeChessDotComPGN(g.PGN)

	moves, err := AnalyzePGN(g, eng, cfg, username)
	if err != nil {
		return models.GameLite{}, err
	}

	g.Moves = moves

	//Uncomment if you want to see games as they are analyzed
	// b, _ := json.MarshalIndent(g, "", "  ")
	// log.Printf("Analysis result: %s", string(b))

	return g, nil
}

func GetMoveAnalysis(color string, before, after models.FENEval) models.MoveAnalysis {
	res := models.MoveAnalysis{}

	// If no CP eval available or we hit a forced mate line, skip classification for now.
	if before.Score.CP == nil || after.Score.CP == nil {
		return res
	}
	if before.Score.Mate != nil || after.Score.Mate != nil {
		return res
	}

	// --- 1) Normalize evals to White's POV ---
	cpBefore := *before.Score.CP
	if before.SideToMove == "b" {
		cpBefore = -cpBefore
	}

	cpAfter := *after.Score.CP
	if after.SideToMove == "b" {
		cpAfter = -cpAfter
	}

	// --- 2) Compute eval change from mover's POV ---
	var delta int
	if color == "w" {
		// White moved → if cpAfter < cpBefore, move was bad for White
		delta = cpAfter - cpBefore
	} else {
		// Black moved → good moves DECREASE cpAfter (because good for Black = bad for White)
		delta = cpBefore - cpAfter
	}

	// If negative, the mover lost good-ness.
	loss := 0
	if delta < 0 {
		loss = -delta
	}
	res.CPChange = loss

	// --- 3) Classify ---
	if loss >= BlunderThreshold {
		res.Is_Blunder = true
	} else if loss >= MistakeThreshold {
		res.Is_Mistake = true
	} else if loss >= InaccuracyThreshold {
		res.Is_Innacuracy = true
	} else if loss >= OpeningInaccuracyThreshold {
		res.Is_Suboptimal = true
	}

	return res
}

// processBatch contains your old main logic for a single batch.
func ProcessBatch(ctx context.Context, cfg *config.Config, job models.JobMessage) error {
	start := time.Now()

	offset := job.BatchIndex * job.NumGames

	log.Printf(
		"Processing batch: user=%s job_id= %s batch_index=%d num_games=%d offset=%d workers=%s",
		job.User, job.JobID, job.BatchIndex, job.NumGames, offset, os.Getenv("WORKERS"),
	)

	games, err := LoadGames(ctx, job.User, job.NumGames, offset)
	if err != nil {
		return err
	}
	if len(games) == 0 {
		log.Printf("no games found for %s (batch_index=%d)", job.User, job.BatchIndex)
		return nil
	}

	numWorkers := GetWorkerCount()
	log.Printf("Analyzing %d games with %d workers", len(games), numWorkers)

	jobs := make(chan models.GameLite, len(games))
	results := make(chan models.GameLite, len(games))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			eng, err := NewUCIEngine(cfg.Engine.Path)
			if err != nil {
				log.Printf("worker %d: failed to create engine: %v", id, err)
				return
			}
			defer eng.Close()
			_ = eng.NewGame()

			for g := range jobs {
				if report, err := AnalyzeOneGame(cfg, eng, g, job.User); err != nil {
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

	// Separate timeout for DB write
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if err := SaveMoves(ctx2, cfg, allResults); err != nil {
		log.Printf("SaveMoves failed for user=%s batch_index=%d: %v", job.User, job.BatchIndex, err)
		return err
	}

	log.Printf(
		"Batch complete: user=%s job_id=%s batch_index=%d num_results=%d took=%s",
		job.User, job.JobID, job.BatchIndex, len(allResults), time.Since(start),
	)

	return nil
}
