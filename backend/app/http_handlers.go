package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"net/http"
	"strings"
	"time"

	"example/my-go-api/app/config"
	"example/my-go-api/app/models"

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

	// Optional: allow ?months=6 (default 3)
	months := 3
	if m := c.Query("months"); m != "" {
		if v, err := parsePositiveInt(m); err == nil && v > 0 && v <= 24 {
			months = v
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()

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

	var out []models.GameLite
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

	// How many games do we keep/save from this endpoint?
	limit := 0
	if q := c.Query("limit"); q != "" {
		if v, err := parsePositiveInt(q); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > 1000 {
		limit = 1000
	}
	if limit <= 0 || limit > len(out) {
		limit = len(out)
	}
	gamesToSave := out[:limit]

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
	jobID, err := CreateJob(ctx, username, totalGames, batchSize, totalBatches)
	if err != nil {
		log.Printf("failed to create job for user=%s: %v", username, err)
		// you can choose to fail the request here if job creation is critical
		// for now we just log and continue
	}

	// ---- enqueue SQS jobs with that jobID ----

	if cfg.QueueURL == "" {
		log.Printf("QUEUE_URL missing in config; skipping enqueue for user=%s", username)
	} else if jobID == "" {
		log.Printf("jobID empty; skipping enqueue for user=%s", username)
	} else {
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			log.Printf("failed to load AWS config for SQS: %v", err)
		} else {
			sqsClient := sqs.NewFromConfig(awsCfg)

			for batchIndex := 0; batchIndex < totalBatches; batchIndex++ {
				jobMsg := models.JobMessage{
					User:       username,
					BatchIndex: batchIndex,
					NumGames:   batchSize,
					JobID:      jobID, // <-- UUID from DB
				}

				body, err := json.Marshal(jobMsg)
				if err != nil {
					log.Printf("failed to marshal JobMessage for user=%s batch=%d: %v",
						username, batchIndex, err)
					continue
				}

				_, err = sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
					QueueUrl:    &cfg.QueueURL,
					MessageBody: aws.String(string(body)),
				})
				if err != nil {
					log.Printf("failed to send SQS message for user=%s batch=%d: %v",
						username, batchIndex, err)
				}
			}
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
