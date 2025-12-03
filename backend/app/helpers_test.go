package app

import (
	"encoding/json"
	"runtime"
	"testing"
	"time"

	"example/my-go-api/app/models"
)

func TestNormalizeChessDotComPGN(t *testing.T) {
	raw := "[Event \"Game\"]\n[Site \"Chess.com\"]\n\n1. e4 {note} e5 1... c5 $5 2. Nf3   2...d6\n"
	got := NormalizeChessDotComPGN(raw)
	want := "1. e4 e5 1... c5 2. Nf3 2...d6"
	if got != want {
		t.Fatalf("NormalizeChessDotComPGN = %q, want %q", got, want)
	}
}

func TestDerivePOV(t *testing.T) {
	var g models.Game
	data := `{"white":{"username":"Alice","result":"win","rating":1500},"black":{"username":"Bob","result":"checkmated","rating":1600}}`
	if err := json.Unmarshal([]byte(data), &g); err != nil {
		t.Fatalf("json.Unmarshal error = %v", err)
	}

	color, opp, rating, result := derivePOV("alice", g)
	if color != "white" || opp != "Bob" || rating != 1600 || result != "win" {
		t.Fatalf("derivePOV unexpected: color=%s opp=%s rating=%d result=%s", color, opp, rating, result)
	}
}

func TestParsePositiveInt(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got, err := parsePositiveInt("42")
		if err != nil || got != 42 {
			t.Fatalf("parsePositiveInt valid = (%d,%v), want (42,nil)", got, err)
		}
	})
	t.Run("invalid", func(t *testing.T) {
		if _, err := parsePositiveInt("not-an-int"); err == nil {
			t.Fatalf("parsePositiveInt should error for invalid input")
		}
	})
}

func TestBuildTagSummary(t *testing.T) {
	tags := map[string]string{
		"White":    "alice",
		"Black":    "bob",
		"WhiteElo": "1500",
		"BlackElo": "1600",
		"Result":   "1-0",
		"Link":     "https://example.test",
	}

	summary := BuildTagSummary(tags, "alice")
	if summary.Color != "white" || summary.Opponent != "bob" || summary.OppRating != 1600 {
		t.Fatalf("BuildTagSummary POV mismatch: %+v", summary)
	}
	if summary.Link != "https://example.test" || summary.Result != "1-0" || summary.WhiteElo != 1500 || summary.BlackElo != 1600 {
		t.Fatalf("BuildTagSummary fields mismatch: %+v", summary)
	}
}

func TestGetUnixTimeStamp(t *testing.T) {
	got := GetUnixTimeStamp("2024.11.22", "12:00:00", "UTC")
	want := time.Date(2024, time.November, 22, 12, 0, 0, 0, time.UTC).Unix()
	if got != want {
		t.Fatalf("GetUnixTimeStamp = %d, want %d", got, want)
	}
}

func TestGetWorkerCount(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("WORKERS", "")
		if got, want := GetWorkerCount(), runtime.NumCPU(); got != want {
			t.Fatalf("GetWorkerCount default = %d, want %d", got, want)
		}
	})

	t.Run("override", func(t *testing.T) {
		t.Setenv("WORKERS", "5")
		if got := GetWorkerCount(); got != 5 {
			t.Fatalf("GetWorkerCount override = %d, want 5", got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Setenv("WORKERS", "not-a-number")
		if got, want := GetWorkerCount(), runtime.NumCPU(); got != want {
			t.Fatalf("GetWorkerCount invalid fallback = %d, want %d", got, want)
		}
	})
}

func TestIsEven(t *testing.T) {
	cases := map[int]bool{0: true, 1: false, 2: true, -1: false}
	for n, expected := range cases {
		if got := IsEven(n); got != expected {
			t.Fatalf("IsEven(%d) = %v, want %v", n, got, expected)
		}
	}
}

func TestNormalizeFEN(t *testing.T) {
	cases := []struct {
		name string
		fen  string
		want string
	}{
		{"basic", "rnbqkbnr/pppp1ppp/8/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq e6 0 2", "rnbqkbnr/pppp1ppp/8/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq e6"},
		{"missing optional", "8/8/8/8/8/8/8/8 b - - 12 34", "8/8/8/8/8/8/8/8 b - -"},
		{"malformed", "too short", "too short"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeFEN(tc.fen); got != tc.want {
				t.Fatalf("NormalizeFEN(%q) = %q, want %q", tc.fen, got, tc.want)
			}
		})
	}
}
