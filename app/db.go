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
	if db == nil {
		// Allow test runs without a backing DB.
		return nil
	}
	if len(games) == 0 {
		return nil
	}

	// One transaction for everything
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1) Temp staging table
	_, err = tx.ExecContext(ctx, `
		CREATE TEMP TABLE tmp_games (
			username         TEXT,
			url              TEXT,
			when_unix        BIGINT,
			color            TEXT,
			opponent         TEXT,
			opponent_rating  INT,
			result           TEXT,
			rated            BOOLEAN,
			time_class       TEXT,
			time_control     TEXT,
			pgn              TEXT
		) ON COMMIT DROP;
	`)
	if err != nil {
		return err
	}

	// 2) COPY into tmp_games
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn(
		"tmp_games",
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

	for _, g := range games {
		if _, err := stmt.Exec(
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
		); err != nil {
			return err
		}
	}

	// finish COPY
	if _, err := stmt.Exec(); err != nil {
		return err
	}
	if err := stmt.Close(); err != nil {
		return err
	}

	// 3) Insert into real table with conflict handling
	_, err = tx.ExecContext(ctx, `
		INSERT INTO games (
			username,
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
		)
		SELECT
			username,
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
		FROM tmp_games
		ON CONFLICT (username, url) DO NOTHING;
	`)
	if err != nil {
		return err
	}

	// 4) Commit
	return tx.Commit()
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
	if db == nil {
		// Allow test runs without a backing DB.
		return nil
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1) Temp staging table
	_, err = tx.ExecContext(ctx, `
		CREATE TEMP TABLE tmp_moves (
			game_id           BIGINT,
			ply               INT,
			move_number       INT,
			fen_before		  TEXT,
			fen_after		  TEXT,
			move_uci          TEXT,
			played_by             CHAR(1),
			eval_before_cp       INT,
			eval_after_cp     INT,
			eval_before_mate         INT,
			eval_after_mate INT,
			eval_depth        INT,
			eval_time      INT,
			centipawn_change INT,
			best_move_uci   TEXT,
			is_inaccuracy BOOLEAN,
			is_mistake BOOLEAN,
			is_blunder BOOLEAN,
			is_suboptimal BOOLEAN,
			normalized_fen_before TEXT
		) ON COMMIT DROP;
	`)
	if err != nil {
		return err
	}

	// 2) COPY into tmp_moves
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn(
		"tmp_moves",
		"game_id", "ply", "move_number", "fen_before", "fen_after",
		"move_uci", "played_by",
		"eval_depth", "eval_time",
		"eval_before_cp", "eval_after_cp",
		"eval_before_mate", "eval_after_mate",
		"centipawn_change", "best_move_uci", "is_inaccuracy", "is_mistake", "is_blunder", "is_suboptimal",
		"normalized_fen_before",
	))
	if err != nil {
		return err
	}

	for _, g := range games {
		for _, e := range g.Moves {

			normalizedFen := NormalizeFEN(e.FenBefore.FEN)
			if _, err := stmt.Exec(
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
				e.Analysis.CPChange,
				e.FenBefore.Score.Best,
				e.Analysis.Is_Innacuracy,
				e.Analysis.Is_Mistake,
				e.Analysis.Is_Blunder,
				e.Analysis.Is_Suboptimal,
				normalizedFen,
			); err != nil {
				return err
			}
		}
	}

	if _, err := stmt.Exec(); err != nil {
		return err
	}
	stmt.Close()

	// 3) Upsert from tmp_moves into moves
	_, err = tx.ExecContext(ctx, `
		INSERT INTO moves (
			game_id, ply, move_number, fen_before,
			fen_after, move_uci, played_by,
			eval_depth, eval_time, eval_before_cp,
			eval_after_cp, eval_before_mate, eval_after_mate, centipawn_change, best_move_uci, is_inaccuracy, is_mistake, is_blunder,
			is_suboptimal, normalized_fen_before
		)
		SELECT
			game_id, ply, move_number, fen_before,
			fen_after, move_uci, played_by,
			eval_depth, eval_time, eval_before_cp,
			eval_after_cp, eval_before_mate, eval_after_mate, centipawn_change, best_move_uci, is_inaccuracy, is_mistake, is_blunder,
			is_suboptimal, normalized_fen_before
		FROM tmp_moves
		ON CONFLICT (game_id, ply) DO NOTHING;
	`)
	if err != nil {
		return err
	}

	return tx.Commit()
}
