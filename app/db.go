package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"example/my-go-api/app/config"
	"example/my-go-api/app/models"

	"github.com/lib/pq"
)

var db *sql.DB

// MustInitDB initializes the global db and panics/logs fatally on error.
func MustInitDB() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s",
		cfg.DB.Username,
		cfg.DB.Password,
		cfg.DB.URL,
		cfg.DB.Port,
	)

	d, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("sql.Open: %v", err)
	}

	if err := d.Ping(); err != nil {
		log.Fatalf("db.Ping: %v", err)
	}

	log.Println("Connected to Postgres")
	db = d
}

func saveGames(ctx context.Context, username string, games []models.GameLite) error {
	if len(games) == 0 {
		return nil
	}

	// One transaction for all games
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// COPY into games table directly
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn(
		"games",
		"username",
		"url",
		"when_unix",
		"color",
		"opponent",
		"opponent_rating",
		"result",
		"rated",
		"time_class",
		"time_control",
		"pgn",
	))
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, g := range games {
		_, err := stmt.Exec(
			username,
			g.URL,
			g.When,
			g.Color,
			g.Opponent,
			g.OppRating,
			g.Result,
			g.Rated,
			g.TimeClass,
			g.TimeControl,
			g.PGN,
		)
		if err != nil {
			return err
		}
	}

	// Finish COPY stream
	if _, err := stmt.Exec(); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// loadGames reads the last N games for a username from the 'games' table
func LoadGames(ctx context.Context, username string, limit int) ([]models.GameLite, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			id,
			url,
			when_unix,
			color,
			opponent,
			opponent_rating,
			result,
			rated,
			time_class,
			time_control,
			pgn
		FROM games
		WHERE username = $1
		ORDER BY when_unix DESC
		LIMIT $2
	`, username, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.GameLite
	for rows.Next() {
		var g models.GameLite
		if err := rows.Scan(
			&g.GameId,
			&g.URL,
			&g.When,
			&g.Color,
			&g.Opponent,
			&g.OppRating,
			&g.Result,
			&g.Rated,
			&g.TimeClass,
			&g.TimeControl,
			&g.PGN,
		); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func SaveMoves(ctx context.Context, cfg *config.Config, games []models.GameLite) error {
	if len(games) == 0 {
		return nil
	}

	// Begin one transaction for all games
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Prepare COPY statement
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn(
		"moves",
		"game_id", "ply", "move_number", "fen_before", "fen_after",
		"move_uci", "played_by",
		"eval_depth", "eval_time",
		"eval_before_cp", "eval_after_cp",
		"eval_before_mate", "eval_after_mate",
		"centipawn_change", "best_move_uci",
	))
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Stream COPY rows
	for _, g := range games {
		for _, e := range g.Moves {

			// safe summed CP
			var summedCP *int
			if e.FenBefore.Score.CP != nil && e.FenAfter.Score.CP != nil {
				v := *e.FenBefore.Score.CP + *e.FenAfter.Score.CP
				summedCP = &v
			}

			_, err := stmt.Exec(
				g.GameId,
				e.Ply,
				e.MoveNumber,
				e.FenBefore.FEN,
				e.FenAfter.FEN,
				e.Move,
				e.Color,
				cfg.Engine.Depth,
				cfg.Engine.MoveTime,
				e.FenBefore.Score.CP,
				e.FenAfter.Score.CP,
				e.FenBefore.Score.Mate,
				e.FenAfter.Score.Mate,
				summedCP,
				e.FenBefore.Score.Best,
			)
			if err != nil {
				return err
			}
		}
	}

	// Close COPY stream
	if _, err := stmt.Exec(); err != nil {
		return err
	}

	// Commit transaction
	return tx.Commit()
}
