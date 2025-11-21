package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"example/my-go-api/app/models"

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
		c.JSON(http.StatusOK, gin.H{"username": username, "games": []models.GameLite{}})
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
			})
		}
	}

	if err := saveGames(ctx, username, out[0:100]); err != nil {
		// For now, just log – you can upgrade this to proper logging later
		fmt.Printf("saveGames failed for %s: %v", username, err)
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"username": username,
		"count":    len(out),
		"games":    out,
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
