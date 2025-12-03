//starts the engine process, speaks UCI over stdin/stdout, and exposes a simple EvalFEN method.

package app

import (
	"bufio"
	"context"
	"errors"
	"example/my-go-api/app/config"
	"example/my-go-api/app/models"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type UCIEngine struct {
	cmd   *exec.Cmd
	in    *bufio.Writer
	out   *bufio.Scanner
	mu    sync.Mutex
	ready bool
}

func NewUCIEngine(path string) (*UCIEngine, error) {
	cmd := exec.Command(path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	e := &UCIEngine{
		cmd: cmd,
		in:  bufio.NewWriter(stdin),
		out: bufio.NewScanner(stdout),
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	// Handshake: "uci" -> wait for "uciok"; also "isready" -> "readyok"
	if err := e.send("uci"); err != nil {
		return nil, err
	}
	for e.out.Scan() {
		line := e.out.Text()
		if line == "uciok" {
			break
		}
	}
	if err := e.send("isready"); err != nil {
		return nil, err
	}
	for e.out.Scan() {
		if e.out.Text() == "readyok" {
			break
		}
	}
	e.ready = true
	return e, nil
}

func (e *UCIEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	_ = e.send("quit")
	return e.cmd.Wait()
}

func (e *UCIEngine) NewGame() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.ready {
		return errors.New("engine not ready")
	}
	if err := e.send("ucinewgame"); err != nil {
		return err
	}
	if err := e.send("isready"); err != nil {
		return err
	}
	for e.out.Scan() {
		if e.out.Text() == "readyok" {
			break
		}
	}
	return nil
}

// EvalFEN evaluates one position. Use either a fixed depth or movetime.
// For beginners, movetime is simple and predictable across hardware.
func (e *UCIEngine) EvalFEN(ctx context.Context, fen string, cfg *config.Config) (models.UCIScore, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.ready {
		return models.UCIScore{}, errors.New("engine not ready")
	}

	// Load position
	if err := e.send(fmt.Sprintf("position fen %s", fen)); err != nil {
		return models.UCIScore{}, err
	}

	if cfg.Engine.DepthOrTime {
		// Analyze using depth
		depth := cfg.Engine.Depth
		if depth <= 0 {
			depth = 12
		}
		if err := e.send(fmt.Sprintf("go depth %d", depth)); err != nil {
			return models.UCIScore{}, err
		}
	} else {
		//analyze using movetime
		if err := e.send(fmt.Sprintf("go movetime %d", cfg.Engine.MoveTime)); err != nil {
			return models.UCIScore{}, err
		}
	}

	var lastScoreCP *int
	var lastScoreMate *int
	var best string

	// Read until "bestmove ..." or context cancels
	readDone := make(chan error, 1)
	go func() {
		for e.out.Scan() {
			line := e.out.Text()
			// Examples we parse:
			// info depth 18 ... score cp 23 ...
			// info depth 20 ... score mate 3 ...
			// bestmove e2e4
			if strings.HasPrefix(line, "info ") {
				if i := strings.Index(line, " score "); i != -1 {
					// score cp N  OR score mate N
					scorePart := line[i+1:]
					if strings.Contains(scorePart, "score cp ") {
						var cp int
						_, _ = fmt.Sscanf(scorePart, "score cp %d", &cp)
						lastScoreCP = &cp
						lastScoreMate = nil
					} else if strings.Contains(scorePart, "score mate ") {
						var m int
						_, _ = fmt.Sscanf(scorePart, "score mate %d", &m)
						lastScoreMate = &m
						lastScoreCP = nil
					}
				}
			} else if strings.HasPrefix(line, "bestmove ") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					best = fields[1]
				}
				break
			}
		}
		readDone <- e.out.Err()
	}()

	var err error
	select {
	case <-ctx.Done():
		_ = e.send("stop")
		select {
		case err = <-readDone:
		case <-time.After(500 * time.Millisecond):
			err = ctx.Err()
		}
	case err = <-readDone:
	}
	if err != nil && err != bufio.ErrBufferFull {
		return models.UCIScore{}, err
	}

	return models.UCIScore{CP: lastScoreCP, Mate: lastScoreMate, Best: best}, nil
}

func (e *UCIEngine) send(cmd string) error {
	_, err := fmt.Fprintln(e.in, cmd)
	if err != nil {
		return err
	}
	return e.in.Flush()
}
