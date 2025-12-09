package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"example/my-go-api/app/config"
)

func newTestEngine(outputLines []string) (*UCIEngine, *strings.Builder) {
	pr, pw := io.Pipe()
	go func() {
		for _, line := range outputLines {
			_, _ = fmt.Fprintln(pw, line)
		}
		_ = pw.Close()
	}()

	var sb strings.Builder
	eng := &UCIEngine{
		in:    bufio.NewWriter(&sb),
		out:   bufio.NewScanner(pr),
		ready: true,
	}
	return eng, &sb
}

func TestEvalFENUsesMovetimeAndParsesScore(t *testing.T) {
	eng, sb := newTestEngine([]string{
		"info depth 10 score cp 23 pv e2e4 e7e5",
		"bestmove e2e4",
	})

	cfg := &config.Config{Engine: config.EngineConfig{DepthOrTime: false, MoveTime: 75}}
	score, err := eng.EvalFEN(context.Background(), "test-fen", cfg)
	if err != nil {
		t.Fatalf("EvalFEN error: %v", err)
	}
	if score.CP == nil || *score.CP != 23 || score.Best != "e2e4" {
		t.Fatalf("EvalFEN unexpected score: %+v", score)
	}

	sent := sb.String()
	if !strings.Contains(sent, "position fen test-fen") {
		t.Fatalf("EvalFEN did not send position command: %q", sent)
	}
	if !strings.Contains(sent, "go movetime 75") {
		t.Fatalf("EvalFEN did not use movetime: %q", sent)
	}
}

func TestEvalFENUsesDepthWhenConfigured(t *testing.T) {
	eng, sb := newTestEngine([]string{"bestmove e2e4"})
	cfg := &config.Config{Engine: config.EngineConfig{DepthOrTime: true}}
	if _, err := eng.EvalFEN(context.Background(), "fen-depth", cfg); err != nil {
		t.Fatalf("EvalFEN error: %v", err)
	}
	if !strings.Contains(sb.String(), "go depth 12") {
		t.Fatalf("EvalFEN should send depth command, got %q", sb.String())
	}
}

func TestEvalFENNotReady(t *testing.T) {
	eng := &UCIEngine{}
	cfg := &config.Config{Engine: config.EngineConfig{MoveTime: 10}}
	if _, err := eng.EvalFEN(context.Background(), "fen", cfg); err == nil {
		t.Fatalf("EvalFEN should fail when engine not ready")
	}
}

func TestNewGameSendsCommands(t *testing.T) {
	pr, pw := io.Pipe()
	go func() {
		_, _ = fmt.Fprintln(pw, "readyok")
		_ = pw.Close()
	}()

	var sb strings.Builder
	eng := &UCIEngine{in: bufio.NewWriter(&sb), out: bufio.NewScanner(pr), ready: true}
	if err := eng.NewGame(); err != nil {
		t.Fatalf("NewGame error: %v", err)
	}
	sent := sb.String()
	if !strings.Contains(sent, "ucinewgame") || !strings.Contains(sent, "isready") {
		t.Fatalf("NewGame did not send expected commands: %q", sent)
	}
}
