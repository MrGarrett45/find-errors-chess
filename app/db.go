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
			color             CHAR(1),
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
			normalized_fen_before TEXT,
			played_by TEXT
		) ON COMMIT DROP;
	`)
	if err != nil {
		return err
	}

	// 2) COPY into tmp_moves
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn(
		"tmp_moves",
		"game_id", "ply", "move_number", "fen_before", "fen_after",
		"move_uci", "color",
		"eval_depth", "eval_time",
		"eval_before_cp", "eval_after_cp",
		"eval_before_mate", "eval_after_mate",
		"centipawn_change", "best_move_uci", "is_inaccuracy", "is_mistake", "is_blunder", "is_suboptimal",
		"normalized_fen_before", "played_by",
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
				e.PlayedBy,
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
			fen_after, move_uci, color,
			eval_depth, eval_time, eval_before_cp,
			eval_after_cp, eval_before_mate, eval_after_mate, centipawn_change, best_move_uci, is_inaccuracy, is_mistake, is_blunder,
			is_suboptimal, normalized_fen_before, played_by
		)
		SELECT
			game_id, ply, move_number, fen_before,
			fen_after, move_uci, color,
			eval_depth, eval_time, eval_before_cp,
			eval_after_cp, eval_before_mate, eval_after_mate, centipawn_change, best_move_uci, is_inaccuracy, is_mistake, is_blunder,
			is_suboptimal, normalized_fen_before, played_by
		FROM tmp_moves
		ON CONFLICT (game_id, ply) DO NOTHING;
	`)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func FindErrorPositions(ctx context.Context, username string) ([]models.SuboptimalFensReport, error) {
	if db == nil {
		return []models.SuboptimalFensReport{}, nil
	}

	const fenQuery = `
WITH user_moves AS (
    SELECT
        m.*,
        g.username,
        g.color        AS game_color,
        g.time_class,
        g.url
    FROM moves m
    JOIN games g ON g.id = m.game_id
    WHERE g.username   = $1
      AND m.played_by  = g.username
      AND m.move_number <= 10
),
position_stats AS (
    SELECT
        normalized_fen_before,
        COUNT(*) AS times_seen,
        SUM(CASE WHEN is_suboptimal THEN 1 ELSE 0 END) AS suboptimal_count,
        SUM(CASE WHEN is_inaccuracy THEN 1 ELSE 0 END) AS inaccuracy_count,
        SUM(CASE WHEN is_mistake    THEN 1 ELSE 0 END) AS mistake_count,
        SUM(CASE WHEN is_blunder    THEN 1 ELSE 0 END) AS blunder_count,
        SUM(
            CASE
                WHEN is_suboptimal
                  OR is_inaccuracy
                  OR is_mistake
                  OR is_blunder
                THEN 1 ELSE 0
            END
        ) AS error_count
    FROM user_moves
    GROUP BY normalized_fen_before
)
SELECT
    normalized_fen_before,
    times_seen,
    suboptimal_count,
    inaccuracy_count,
    mistake_count,
    blunder_count,
    error_count,
    (error_count::float / times_seen) AS error_rate
FROM position_stats
WHERE times_seen  >= 3
  AND error_count >= 2
ORDER BY error_rate DESC, times_seen DESC;
`

	rows, err := db.QueryContext(ctx, fenQuery, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []models.SuboptimalFensReport
	for rows.Next() {
		var fen models.SuboptimalFen
		var blunderCount int
		if err := rows.Scan(
			&fen.NormalizedFenBefore,
			&fen.TimesSeen,
			&fen.SuboptimalCount,
			&fen.InaccuracyCount,
			&fen.MistakeCount,
			&blunderCount,
			&fen.ErrorCount,
			&fen.ErrorRate,
		); err != nil {
			return nil, err
		}

		moves, err := fetchErrorMoves(ctx, username, fen.NormalizedFenBefore)
		if err != nil {
			return nil, err
		}

		reports = append(reports, models.SuboptimalFensReport{
			BadFen: fen,
			Moves:  moves,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reports, nil
}

func fetchErrorMoves(ctx context.Context, username, normalizedFen string) ([]models.Move, error) {
	const movesQuery = `
SELECT
    g.username,
    g.url,
    g.when_unix,
    g.color            AS game_color,
    g.opponent,
    g.opponent_rating,
    g.result,
    g.time_class,
    g.time_control,

    m.game_id,
    m.ply,
    m.move_number,
    m.color            AS move_color,
    m.normalized_fen_before,
    m.fen_before,

    m.move_san,
    m.move_uci         AS played_move_uci,
    m.best_move_uci    AS engine_best_move_uci,

    m.eval_before_cp,
    m.eval_after_cp,
    m.centipawn_change,

    m.is_suboptimal,
    m.is_inaccuracy,
    m.is_mistake,
    m.is_blunder
FROM moves m
JOIN games g ON g.id = m.game_id
WHERE g.username          = $1
  AND m.played_by         = g.username
  AND m.normalized_fen_before = $2
  AND (
        m.is_suboptimal
     OR m.is_inaccuracy
     OR m.is_mistake
     OR m.is_blunder
  )
ORDER BY g.when_unix DESC;
`

	rows, err := db.QueryContext(ctx, movesQuery, username, normalizedFen)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var moves []models.Move
	for rows.Next() {
		var (
			user        string
			url         string
			whenUnix    int64
			gameColor   string
			opponent    string
			opponentElo int
			result      string
			timeClass   string
			timeControl string

			gameID         int64
			ply            int
			moveNumber     int
			moveColor      string
			normalized     string
			fenBefore      string
			moveSAN        sql.NullString
			playedMoveUCI  string
			engineBestMove sql.NullString
			evalBeforeCP   sql.NullInt64
			evalAfterCP    sql.NullInt64
			cpChange       sql.NullInt64
			isSuboptimal   bool
			isInaccuracy   bool
			isMistake      bool
			isBlunder      bool
		)

		if err := rows.Scan(
			&user,
			&url,
			&whenUnix,
			&gameColor,
			&opponent,
			&opponentElo,
			&result,
			&timeClass,
			&timeControl,
			&gameID,
			&ply,
			&moveNumber,
			&moveColor,
			&normalized,
			&fenBefore,
			&moveSAN,
			&playedMoveUCI,
			&engineBestMove,
			&evalBeforeCP,
			&evalAfterCP,
			&cpChange,
			&isSuboptimal,
			&isInaccuracy,
			&isMistake,
			&isBlunder,
		); err != nil {
			return nil, err
		}

		// quiet unused context fields that are not mapped onto the Move struct
		_ = whenUnix
		_ = gameColor
		_ = opponentElo
		_ = result
		_ = timeClass
		_ = timeControl
		_ = gameID
		_ = normalized
		_ = evalAfterCP

		mv := models.Move{
			Move:       playedMoveUCI,
			PlayedBy:   user,
			MoveNumber: moveNumber,
			Ply:        ply,
			Color:      moveColor,
			FenBefore: models.FENEval{
				MoveNumber: moveNumber,
				SideToMove: moveColor,
				FEN:        fenBefore,
				Score: models.UCIScore{
					CP:   nullableIntToPtr(evalBeforeCP),
					Best: engineBestMove.String,
				},
			},
			Analysis: models.MoveAnalysis{
				CPChange:      int(cpChange.Int64),
				Is_Suboptimal: isSuboptimal,
				Is_Innacuracy: isInaccuracy,
				Is_Mistake:    isMistake,
				Is_Blunder:    isBlunder,
			},
			URL:      url,
			Opponent: opponent,
		}

		moves = append(moves, mv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return moves, nil
}

func nullableIntToPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}
