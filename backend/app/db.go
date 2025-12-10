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
			pgn              TEXT,
			eco              TEXT
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
		"eco",
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
			g.ECO,
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
			pgn,
			eco
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
			pgn,
			eco
		FROM tmp_games
		ON CONFLICT (username, url) DO NOTHING;
	`)
	if err != nil {
		return err
	}

	// 4) Commit
	return tx.Commit()
}

// LoadGames reads a batch of games for a username using LIMIT/OFFSET.
// Example: limit = 100, offset = batchIndex * limit
func LoadGames(ctx context.Context, username string, limit, offset int) ([]models.GameLite, error) {
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
		OFFSET $3
	`, username, limit, offset)
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
        ) AS error_count,
        MIN(color) AS side_to_move
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
    (error_count::float / times_seen) AS error_rate,
    side_to_move
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

	var (
		reports []models.SuboptimalFensReport
		fens    []string
	)

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
			&fen.SideToMove,
		); err != nil {
			return nil, err
		}

		// collect FEN list for batch query
		fens = append(fens, fen.NormalizedFenBefore)

		reports = append(reports, models.SuboptimalFensReport{
			BadFen: fen,
			Moves:  nil, // will fill after batch query
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(reports) == 0 {
		return reports, nil
	}

	// batch fetch moves for all FENs at once
	movesByFEN, err := fetchErrorMovesBatch(ctx, username, fens)
	if err != nil {
		return nil, err
	}

	// attach moves to each report
	for i := range reports {
		fenKey := reports[i].BadFen.NormalizedFenBefore
		if mv, ok := movesByFEN[fenKey]; ok {
			reports[i].Moves = mv
		} else {
			// keep as empty slice if no moves found
			reports[i].Moves = []models.Move{}
		}
	}

	return reports, nil
}

// Thin wrapper to preserve existing API if other code calls this.
// func fetchErrorMoves(ctx context.Context, username, normalizedFen string) ([]models.Move, error) {
// 	movesByFEN, err := fetchErrorMovesBatch(ctx, username, []string{normalizedFen})
// 	if err != nil {
// 		return nil, err
// 	}
// 	return movesByFEN[normalizedFen], nil
// }

// New batched helper: fetches error moves for many FENs in one query.
func fetchErrorMovesBatch(ctx context.Context, username string, normalizedFens []string) (map[string][]models.Move, error) {
	result := make(map[string][]models.Move, len(normalizedFens))
	if len(normalizedFens) == 0 {
		return result, nil
	}

	const movesQuery = `
SELECT
    g.username,
    g.url,
	g.eco,
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
WHERE g.username              = $1
  AND m.played_by             = g.username
  AND m.normalized_fen_before = ANY($2)
  AND (
        m.is_suboptimal
     OR m.is_inaccuracy
     OR m.is_mistake
     OR m.is_blunder
  )
ORDER BY m.normalized_fen_before, g.when_unix DESC;
`

	rows, err := db.QueryContext(ctx, movesQuery, username, pq.Array(normalizedFens))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			user        string
			url         string
			eco         string
			whenUnix    int64
			gameColor   string
			opponent    string
			opponentElo int
			resultStr   string
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
			&eco,
			&whenUnix,
			&gameColor,
			&opponent,
			&opponentElo,
			&resultStr,
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
		_ = resultStr
		_ = timeClass
		_ = timeControl
		_ = gameID
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
			ECO:      eco,
			Opponent: opponent,
		}

		result[normalized] = append(result[normalized], mv)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// unchanged helper
func nullableIntToPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}

func CreateJob(ctx context.Context, username string, totalGames, batchSize, totalBatches int) (string, error) {
	const q = `
        INSERT INTO jobs (username, total_games, batch_size, total_batches)
        VALUES ($1, $2, $3, $4)
        RETURNING id;
    `
	var jobID string
	if err := db.QueryRowContext(ctx, q, username, totalGames, batchSize, totalBatches).Scan(&jobID); err != nil {
		return "", err
	}
	log.Printf("Created job %s for user=%s totalGames=%d totalBatches=%d", jobID, username, totalGames, totalBatches)
	return jobID, nil
}

// UpdateJobProgress increments completed_batches for a job and sets
// status to 'running' or 'completed' accordingly.
func UpdateJobProgress(ctx context.Context, jobID string) error {
	const q = `
        UPDATE jobs
        SET
            completed_batches = completed_batches + 1,
            status = CASE
                WHEN completed_batches + 1 >= total_batches THEN 'completed'
                ELSE 'running'
            END,
            updated_at = now()
        WHERE id = $1;
    `

	res, err := db.ExecContext(ctx, q, jobID)
	if err != nil {
		return err
	}

	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		log.Printf("UpdateJobProgress: no job row found for id=%s", jobID)
	}

	return nil
}

// FindJobStatus fetches status and batch counts for a job id.
func FindJobStatus(ctx context.Context, jobID string) (models.JobStatus, error) {
	var js models.JobStatus

	const q = `
        SELECT id, status, completed_batches, total_batches
        FROM jobs
        WHERE id = $1;
    `

	row := db.QueryRowContext(ctx, q, jobID)
	if err := row.Scan(&js.ID, &js.Status, &js.CompletedBatches, &js.TotalBatches); err != nil {
		if err == sql.ErrNoRows {
			return models.JobStatus{}, err
		}
		return models.JobStatus{}, err
	}

	return js, nil
}
