package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"example/my-go-api/app/config"
	"example/my-go-api/app/models"

	_ "github.com/lib/pq"
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

	//"postgres://postgres:lTCXTLfzEnvvU8QOfpAb@lTCXTLfzEnvvU8QOfpAb@chess-db.crslkdnd8iac.us-east-1.rds.amazonaws.com:5432"
	//"postgres://postgres:lTCXTLfzEnvvU8QOfpAb@chess-db.crslkdnd8iac.us-east-1.rds.amazonaws.com:5432"
	log.Println("Connected to Postgres")
	db = d
}

func saveGames(ctx context.Context, username string, games []models.GameLite) error {
	if len(games) == 0 {
		return nil
	}

	// Wrap in a transaction so we either write all or none
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO games (
			username, url, when_unix, color, opponent, opponent_rating,
			result, rated, time_class, time_control, pgn
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (username, url) DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, g := range games {
		_, err := stmt.ExecContext(
			ctx,
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

func SaveMoves(ctx context.Context, cfg *config.Config, g models.GameLite) error {
	if len(g.Moves) == 0 {
		return nil
	}

	// Wrap in a transaction so we either write all or none
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO moves (
			game_id, ply, move_number, fen_before, fen_after, move_uci, played_by, eval_depth, eval_time,
			eval_before_cp, eval_after_cp, eval_before_mate, eval_after_mate, centipawn_change, best_move_uci
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (game_id, ply) DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range g.Moves {
		// compute summed CP safely (or nil)
		var summedCP *int
		if e.FenBefore.Score.CP != nil && e.FenAfter.Score.CP != nil {

			v := *e.FenBefore.Score.CP + *e.FenAfter.Score.CP
			summedCP = &v
		} else {
			// either score/CP is nil; store NULL in DB
			summedCP = nil
		}
		_, err := stmt.ExecContext(
			ctx,
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

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
