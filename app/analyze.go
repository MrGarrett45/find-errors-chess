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

const (
	InaccuracyThreshold        = 50  // 0.50 pawns
	MistakeThreshold           = 100 // 1.00 pawns
	BlunderThreshold           = 200 // 2.00 pawns
	OpeningInaccuracyThreshold = 30
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

		moveAnalysis := GetMoveAnalysis(color, fens[i], fenAfter)
		moves = append(moves, models.Move{Move: m.String(), FenBefore: fens[i], FenAfter: fenAfter, MoveNumber: moveNumber, Ply: i + 1, Color: color, Analysis: moveAnalysis})
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
