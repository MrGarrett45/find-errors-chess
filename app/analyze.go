// --- analyze.go ---
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"example/my-go-api/app/config"
	"example/my-go-api/app/models"

	"github.com/notnil/chess"
)

func AnalyzePGN(meta models.GameLite, eng *UCIEngine, cfg *config.Config) ([]models.Move, error) {
	// Parse PGN into new game
	g := chess.NewGame()
	if err := g.UnmarshalText([]byte(meta.PGN)); err != nil {
		return []models.Move{}, err
	}

	// Collect FEN snapshots of each position
	var fens []models.FENEval
	for i, p := range g.Positions() {
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

		moveNumber := (i / 2) + 1

		//make sure fenAfter exists for any given position before using it
		var fenAfter models.FENEval
		if i+1 < len(fens) {
			fenAfter = fens[i+1]
		} else {
			fenAfter = models.FENEval{}
		}

		moves = append(moves, models.Move{Move: m.String(), FenBefore: fens[i], FenAfter: fenAfter, MoveNumber: moveNumber, Ply: i + 1, Color: color})
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
func AnalyzeOneGame(cfg *config.Config, eng *UCIEngine, g models.GameLite) (models.GameLite, error) {
	fmt.Printf("\n=== Analyzing game: %s vs %s (%s) ===\n", cfg.User, g.Opponent, g.URL)

	//tags := ParsePGNTags(g.PGN)
	g.PGN = NormalizeChessDotComPGN(g.PGN)

	moves, err := AnalyzePGN(g, eng, cfg)
	if err != nil {
		return models.GameLite{}, err
	}

	g.Moves = moves

	b, _ := json.MarshalIndent(g, "", "  ")
	fmt.Println(string(b))

	return g, nil
}
