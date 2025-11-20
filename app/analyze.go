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

func AnalyzePGN(pgn string, meta models.GameLite, eng *UCIEngine, cfg *config.Config) (models.GameEval, error) {
	// Parse PGN into new game
	g := chess.NewGame()
	if err := g.UnmarshalText([]byte(pgn)); err != nil {
		return models.GameEval{}, err
	}

	// Collect FEN snapshots of each position
	var fens []models.PositionEval
	for i, p := range g.Positions() {
		if i >= cfg.Engine.NumMoves {
			break
		}

		if i > 0 {
			fenEval := fenEvalFromPosition(p)

			// Assign move in SAN/UCN
			if i < len(g.Moves()) {
				fenEval.Move = g.Moves()[i].String()
			}

			// Assign move number in chess notation (1, 1, 2, 2, ...)
			fenEval.MoveNumber = ((i - 1) / 2) + 1
			fenEval.Ply = i

			fens = append(fens, fenEval)
		}

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

	//this is inverted for some reason
	side := "b"
	if pos.Turn() == chess.Black {
		side = "w"
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

// What we let our workers call to process games
func AnalyzeOneGame(cfg *config.Config, eng *UCIEngine, g models.GameLite) (models.GameEval, error) {
	fmt.Printf("\n=== Analyzing game: %s vs %s (%s) ===\n", cfg.User, g.Opponent, g.URL)

	denormalizedPgn := g.PGN

	tags := ParsePGNTags(denormalizedPgn)
	pgn := NormalizeChessDotComPGN(denormalizedPgn)

	meta := models.GameLite{
		URL:         tags["Link"],
		When:        GetUnixTimeStamp(tags["Date"], tags["StartTime"], tags["Timezone"]),
		Color:       "white",
		Opponent:    "demo",
		Result:      tags["Result"],
		PGN:         pgn,
		TimeControl: tags["TimeControl"],
	}

	report, err := AnalyzePGN(meta.PGN, meta, eng, cfg)
	if err != nil {
		return models.GameEval{}, err
	}

	b, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(b))
	return report, nil
}
