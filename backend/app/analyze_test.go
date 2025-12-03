package app

import (
	"strings"
	"testing"

	"example/my-go-api/app/models"
	"github.com/notnil/chess"
)

func intPtr(v int) *int { return &v }

func TestFenInfoFromPosition(t *testing.T) {
	g := chess.NewGame()

	start := fenInfoFromPosition(g.Position())
	if start.SideToMove != "w" || start.MoveNumber != 1 {
		t.Fatalf("start fen info = %+v", start)
	}
	if !strings.HasPrefix(start.FEN, "rnbqkbnr/pppppppp") {
		t.Fatalf("unexpected start FEN: %s", start.FEN)
	}

	if err := g.MoveStr("e4"); err != nil {
		t.Fatalf("move e4: %v", err)
	}
	if err := g.MoveStr("c5"); err != nil {
		t.Fatalf("move c5: %v", err)
	}

	after := fenInfoFromPosition(g.Position())
	if after.SideToMove != "w" || after.MoveNumber != 2 {
		t.Fatalf("after fen info = %+v", after)
	}
}

func TestGetMoveAnalysisWhiteBlunder(t *testing.T) {
	before := models.FENEval{SideToMove: "w", Score: models.UCIScore{CP: intPtr(50)}}
	after := models.FENEval{SideToMove: "b", Score: models.UCIScore{CP: intPtr(150)}}

	res := GetMoveAnalysis("w", before, after)
	if !res.Is_Blunder || res.CPChange != 200 {
		t.Fatalf("expected blunder with loss 200, got %+v", res)
	}
	if res.Is_Mistake || res.Is_Innacuracy || res.Is_Suboptimal {
		t.Fatalf("only blunder should be flagged, got %+v", res)
	}
}

func TestGetMoveAnalysisBlackMistake(t *testing.T) {
	before := models.FENEval{SideToMove: "b", Score: models.UCIScore{CP: intPtr(0)}}
	after := models.FENEval{SideToMove: "w", Score: models.UCIScore{CP: intPtr(120)}}

	res := GetMoveAnalysis("b", before, after)
	if !res.Is_Mistake || res.CPChange != 120 {
		t.Fatalf("expected mistake with loss 120, got %+v", res)
	}
	if res.Is_Blunder || res.Is_Innacuracy || res.Is_Suboptimal {
		t.Fatalf("only mistake should be flagged, got %+v", res)
	}
}

func TestGetMoveAnalysisSkipsMateLines(t *testing.T) {
	mate := 3
	before := models.FENEval{SideToMove: "w", Score: models.UCIScore{Mate: &mate}}
	after := models.FENEval{SideToMove: "b", Score: models.UCIScore{CP: intPtr(0)}}

	res := GetMoveAnalysis("w", before, after)
	if res.Is_Blunder || res.Is_Mistake || res.Is_Innacuracy || res.Is_Suboptimal || res.CPChange != 0 {
		t.Fatalf("expected empty analysis when mate present, got %+v", res)
	}
}
