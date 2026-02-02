package app

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"example/my-go-api/app/config"
	"example/my-go-api/app/models"
	"example/my-go-api/auth"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
)

var httpc = &http.Client{Timeout: 15 * time.Second}

type archiveIndex struct {
	Archives []string `json:"archives"`
}

type monthlyGames struct {
	Games []models.Game `json:"games"`
}

func GetChessGames(c *gin.Context) {
	username := strings.ToLower(c.Param("username"))
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing username"})
		return
	}

	claims, ok := auth.ClaimsFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing auth context"})
		return
	}

	// Optional: allow ?months=6 (default 3)
	months := 3
	if m := c.Query("months"); m != "" {
		if v, err := parsePositiveInt(m); err == nil && v > 0 && v <= 24 {
			months = v
		}
	}

	// Optional: limit games to save/process
	limit := 0
	if q := c.Query("limit"); q != "" {
		if v, err := parsePositiveInt(q); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > 1000 {
		limit = 1000
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()

	var out []models.GameLite
	provider := strings.ToLower(strings.TrimSpace(c.Query("provider")))
	if provider == "lichess" {
		lichessGames, err := fetchLichessGames(ctx, username, months, limit)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, errUserNotFound) {
				status = http.StatusNotFound
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		out = lichessGames
	} else {
		archives, err := fetchArchives(ctx, username)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, errUserNotFound) {
				status = http.StatusNotFound
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		if len(archives) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"username": username,
				"count":    0,
			})
			return
		}

		// Take last N months (archives are chronological)
		start := len(archives) - months
		if start < 0 {
			start = 0
		}
		target := archives[start:]

		for i := len(target) - 1; i >= 0; i-- { // newest first
			monthURL := target[i]
			mg, err := fetchMonthly(ctx, monthURL)
			if err != nil {
				// soft-fail a month; you could also collect and return partial errors
				continue
			}
			for _, g := range mg.Games {
				color, opp, oppRating, result := derivePOV(username, g)
				eco := NormalizeECO(g.ECO)
				out = append(out, models.GameLite{
					URL:         g.URL,
					When:        g.EndTime,
					Color:       color,
					Opponent:    opp,
					OppRating:   oppRating,
					Result:      result,
					Rated:       g.Rated,
					TimeClass:   g.TimeClass,
					TimeControl: g.TimeControl,
					PGN:         g.PGN,
					ECO:         eco,
				})
			}
		}
	}

	if len(out) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"username": username,
			"count":    0,
		})
		return
	}

	// ---- load config for limits + queue URL ----

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("LoadConfig failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load config"})
		return
	}
	engineSettings := models.EngineSettings{}
	if v := c.Query("engine_depth"); v != "" {
		if d, err := parsePositiveInt(v); err == nil {
			engineSettings.Depth = d
		}
	}
	if v := c.Query("engine_move_time"); v != "" {
		if mt, err := parsePositiveInt(v); err == nil {
			engineSettings.MoveTimeMS = mt
		}
	}
	if v := c.Query("engine_depth_or_time"); v != "" {
		if useDepth, err := strconv.ParseBool(v); err == nil {
			engineSettings.UseDepth = useDepth
		}
	}

	if limit <= 0 || limit > len(out) {
		limit = len(out)
	}
	gamesToSave := out[:limit]

	_, err = enforceWeeklyQuota(c.Request.Context(), claims.Subject, len(gamesToSave))
	if err != nil {
		if qe, ok := err.(quotaError); ok {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":        "quota_exceeded",
				"message":      "Free users can analyze up to 100 games per week.",
				"limit":        qe.Limit,
				"analysesUsed": qe.Used,
			})
			return
		}
		log.Printf("failed to enforce quota: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify quota"})
		return
	}

	// Save games
	if err := saveGames(ctx, username, gamesToSave); err != nil {
		log.Printf("saveGames failed for %s: %v", username, err)
		// not fatal for the endpoint, we still return a 200 w/ games
	}

	// ---- compute batches and create a job row ----

	batchSize := cfg.Engine.NumGames // games per worker/batch
	if batchSize <= 0 {
		batchSize = 100 // sane fallback
	}

	totalGames := limit
	totalBatches := (totalGames + batchSize - 1) / batchSize // ceil division

	// Record that a job has begun
	user, err := getUserByAuth0Sub(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = UpsertUserFromClaims(ctx, claims)
			user, err = getUserByAuth0Sub(ctx, claims.Subject)
		}
		if err != nil {
			log.Printf("failed to resolve user id for sub=%s: %v", claims.Subject, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin analysis"})
			return
		}
	}

	jobID, err := CreateJob(ctx, username, user.ID, totalGames, batchSize, totalBatches)
	if err != nil {
		log.Printf("failed to create job for user=%s: %v", username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin analysis"})
		return
	}

	// ---- enqueue SQS jobs with that jobID ----

	if cfg.QueueURL == "" {
		log.Printf("QUEUE_URL missing in config; cannot enqueue for user=%s", username)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin analysis"})
		return
	}
	if jobID == "" {
		log.Printf("jobID empty; skipping enqueue for user=%s", username)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin analysis"})
		return
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("failed to load AWS config for SQS: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin analysis"})
		return
	}

	sqsClient := sqs.NewFromConfig(awsCfg)
	for batchIndex := 0; batchIndex < totalBatches; batchIndex++ {
		jobMsg := models.JobMessage{
			User:           username,
			BatchIndex:     batchIndex,
			NumGames:       batchSize,
			JobID:          jobID, // <-- UUID from DB
			EngineDepth:    engineSettings.Depth,
			EngineMoveTime: engineSettings.MoveTimeMS,
			EngineUseDepth: engineSettings.UseDepth,
		}

		body, err := json.Marshal(jobMsg)
		if err != nil {
			log.Printf("failed to marshal JobMessage for user=%s batch=%d: %v",
				username, batchIndex, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin analysis"})
			return
		}

		_, err = sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
			QueueUrl:    &cfg.QueueURL,
			MessageBody: aws.String(string(body)),
		})
		if err != nil {
			log.Printf("failed to send SQS message for user=%s batch=%d: %v",
				username, batchIndex, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin analysis"})
			return
		}
	}

	// ---- Response: send back the games we actually saved/are processing ----
	c.IndentedJSON(http.StatusOK, gin.H{
		"username": username,
		"count":    len(gamesToSave),
		"job_id":   jobID,
		"batches":  totalBatches,
	})
}

func fetchLichessGames(ctx context.Context, username string, months, maxGames int) ([]models.GameLite, error) {
	base := fmt.Sprintf("https://lichess.org/api/games/user/%s", url.PathEscape(username))
	params := url.Values{}
	params.Set("pgnInJson", "true")
	params.Set("moves", "true")
	params.Set("tags", "true")
	params.Set("opening", "true")
	params.Set("clocks", "true")
	params.Set("sort", "dateDesc")
	since := time.Now().AddDate(0, -months, 0).UnixMilli()
	params.Set("since", strconv.FormatInt(since, 10))
	if maxGames > 0 {
		params.Set("max", strconv.Itoa(maxGames))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/x-ndjson")

	res, err := httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil, errUserNotFound
	}
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return nil, fmt.Errorf("lichess api error: %s", strings.TrimSpace(string(body)))
	}

	scanner := bufio.NewScanner(res.Body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 2*1024*1024)

	var out []models.GameLite
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var g models.LichessGame
		if err := json.Unmarshal(line, &g); err != nil {
			continue
		}
		game, ok := mapLichessGame(username, g)
		if ok {
			out = append(out, game)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func mapLichessGame(username string, g models.LichessGame) (models.GameLite, bool) {
	user := strings.ToLower(username)
	whiteName := lichessPlayerName(g.Players.White)
	blackName := lichessPlayerName(g.Players.Black)
	if whiteName == "" || blackName == "" {
		return models.GameLite{}, false
	}

	color := "black"
	opponent := whiteName
	oppRating := g.Players.White.Rating
	if strings.ToLower(whiteName) == user {
		color = "white"
		opponent = blackName
		oppRating = g.Players.Black.Rating
	}

	whenMs := g.LastMoveAt
	if whenMs == 0 {
		whenMs = g.CreatedAt
	}
	whenUnix := whenMs / 1000

	timeClass := g.Speed
	if timeClass == "" {
		timeClass = g.Perf
	}

	timeControl := ""
	if g.Clock != nil {
		timeControl = fmt.Sprintf("%d+%d", g.Clock.Initial, g.Clock.Increment)
	}

	eco := ""
	if g.Opening != nil {
		if g.Opening.Name != "" {
			eco = g.Opening.Name
		} else {
			eco = g.Opening.ECO
		}
	}

	return models.GameLite{
		URL:         fmt.Sprintf("https://lichess.org/%s", g.ID),
		When:        whenUnix,
		Color:       color,
		Opponent:    opponent,
		OppRating:   oppRating,
		Result:      lichessResultForUser(g.Winner, color),
		Rated:       g.Rated,
		TimeClass:   timeClass,
		TimeControl: timeControl,
		PGN:         g.PGN,
		ECO:         eco,
	}, true
}

func lichessPlayerName(p models.LichessPlayer) string {
	if p.User != nil && p.User.Name != "" {
		return p.User.Name
	}
	return p.Name
}

func lichessResultForUser(winner string, color string) string {
	if winner == "" {
		return "draw"
	}
	if (winner == "white" && color == "white") || (winner == "black" && color == "black") {
		return "win"
	}
	return "loss"
}

// GetErrorPositions returns a slice of error positions for the given user.
// It relies on a db function that will be implemented later.
func GetErrorPositions(c *gin.Context) {
	username := strings.ToLower(c.Param("username"))
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing username"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	positions, err := FindErrorPositions(ctx, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"username":  username,
		"count":     len(positions),
		"positions": positions,
	})
}

// GetGamesCount returns a count of stored games for a user.
func GetGamesCount(c *gin.Context) {
	username := strings.ToLower(c.Param("username"))
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing username"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	count, err := CountGames(ctx, username)
	if err != nil {
		log.Printf("count games failed for %s: %v", username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count games"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"username": username,
		"count":    count,
	})
}

// GetJobStatus returns status and batch progress for a job.
func GetJobStatus(c *gin.Context) {
	jobID := c.Param("jobid")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing job id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	status, err := FindJobStatus(ctx, jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job": status,
	})
}

var errUserNotFound = errors.New("user not found")

func fetchArchives(ctx context.Context, username string) ([]string, error) {
	u := fmt.Sprintf("https://api.chess.com/pub/player/%s/games/archives", username)
	var idx archiveIndex
	if err := getJSON(ctx, u, &idx); err != nil {
		if httpErr, ok := err.(httpError); ok && httpErr.Status == http.StatusNotFound {
			return nil, errUserNotFound
		}
		return nil, err
	}
	return idx.Archives, nil
}

func fetchMonthly(ctx context.Context, monthURL string) (*monthlyGames, error) {
	var mg monthlyGames
	if err := getJSON(ctx, monthURL, &mg); err != nil {
		return nil, err
	}
	return &mg, nil
}

type httpError struct {
	Status int
	Body   string
}

func (e httpError) Error() string { return fmt.Sprintf("http %d: %s", e.Status, e.Body) }

func getJSON(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	// Friendly UA per Chess.com guidelines
	req.Header.Set("User-Agent", "MyChessReview/0.1 (contact: garrettmclaughlin1980@gmail.com)")

	// basic retry for 429/5xx
	var last httpError
	for attempt := 0; attempt < 3; attempt++ {
		res, err := httpc.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		if res.StatusCode == http.StatusOK {
			return json.NewDecoder(res.Body).Decode(v)
		}

		// capture body (truncated) for error clarity
		var msg struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(res.Body).Decode(&msg)
		last = httpError{Status: res.StatusCode, Body: msg.Message}

		if res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500 {
			time.Sleep(time.Duration(250*(attempt+1)) * time.Millisecond)
			continue
		}
		break
	}
	return last
}
