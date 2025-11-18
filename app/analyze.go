// --- analyze.go ---
package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"example/my-go-api/app/models"

	"github.com/notnil/chess"
)

func AnalyzePGN(pgn string, meta models.GameLite, eng *UCIEngine) (models.GameEval, error) {
	// Parse PGN into new game
	g := chess.NewGame()
	if err := g.UnmarshalText([]byte(pgn)); err != nil {
		return models.GameEval{}, err
	}

	// Collect FEN snapshots of each position
	var fens []models.PositionEval
	for _, p := range g.Positions() {
		fens = append(fens, fenEvalFromPosition(p))
	}

	// Collect individual moves for easier debugging
	for i, m := range g.Moves() {
		fens[i].Move = m.String()
	}

	// New game (lets the engine clear its internal state)
	eng.NewGame()

	// Evaluate each picked FEN with ~500ms each (tweakable)
	ctx := context.Background()
	for i := range fens {
		c2, cancel := context.WithTimeout(ctx, 2*time.Second)
		score, _ := eng.EvalFEN(c2, fens[i].FEN, 500)
		cancel()
		fens[i].Score = score
	}

	// Final summary = final position eval if available
	summary := models.UCIScore{}
	if len(fens) > 0 {
		summary = fens[len(fens)-1].Score
	}

	return models.GameEval{
		URL:      meta.URL,
		When:     meta.When,
		Color:    meta.Color,
		Opponent: meta.Opponent,
		Result:   meta.Result,
		Evals:    fens,
		Summary:  summary,
	}, nil
}

func fenEvalFromPosition(pos *chess.Position) models.PositionEval {
	fen := pos.String()
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
	return models.PositionEval{
		MoveNumber: moveNum,
		SideToMove: side,
		FEN:        string(fen),
	}
}
