package app

import (
	"example/my-go-api/app/models"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type TagSummary struct {
	Event       string `json:"event,omitempty"`
	Site        string `json:"site,omitempty"`
	Date        string `json:"date,omitempty"` // "YYYY.MM.DD"
	Round       string `json:"round,omitempty"`
	Result      string `json:"result,omitempty"` // "1-0","0-1","1/2-1/2","*"
	White       string `json:"white,omitempty"`
	Black       string `json:"black,omitempty"`
	WhiteElo    int    `json:"white_elo,omitempty"`
	BlackElo    int    `json:"black_elo,omitempty"`
	TimeControl string `json:"time_control,omitempty"` // "600" or "600+0"
	Termination string `json:"termination,omitempty"`
	Link        string `json:"link,omitempty"`
	ECO         string `json:"eco,omitempty"`
	ECOUrl      string `json:"eco_url,omitempty"`
	UTCDate     string `json:"utc_date,omitempty"` // "YYYY.MM.DD"
	UTCTime     string `json:"utc_time,omitempty"` // "HH:MM:SS"
	CurrentFEN  string `json:"current_fen,omitempty"`
	// POV helpers (if you pass your username)
	Color     string `json:"color,omitempty"` // "white" or "black" (for povUser)
	Opponent  string `json:"opponent,omitempty"`
	OppRating int    `json:"opponent_rating,omitempty"`
}

var (
	reTags     = regexp.MustCompile(`(?m)^\[.*?\]\s*`) // [Tag "Value"] lines
	reComments = regexp.MustCompile(`\{[^}]*\}`)       // {...} comments (incl. [%clk ...])
	reNAG      = regexp.MustCompile(`\$\d+`)           // $1, $2, etc.
	reSpaces   = regexp.MustCompile(`\s+`)
	reEcoMoves = regexp.MustCompile(`-\d.*`)
)

// layout for unix timestamp conversion
var layout = "2006.01.02 15:04:05"

// NormalizeChessDotComPGN removes headers/comments/NAGs and collapses whitespace.
func NormalizeChessDotComPGN(pgn string) string {
	pgn = reTags.ReplaceAllString(pgn, "")
	pgn = reComments.ReplaceAllString(pgn, "")
	pgn = reNAG.ReplaceAllString(pgn, "")
	pgn = reSpaces.ReplaceAllString(strings.TrimSpace(pgn), " ")
	return pgn
}

func derivePOV(username string, g models.Game) (color, opponent string, oppRating int, result string) {
	u := strings.ToLower(username)
	if strings.ToLower(g.White.Username) == u {
		return "white", g.Black.Username, g.Black.Rating, g.White.Result
	}
	return "black", g.White.Username, g.White.Rating, g.Black.Result
}

// NormalizeECO turns an ECO URL or slug into a readable opening name without move suffixes.
func NormalizeECO(ecoURL string) string {
	ecoURL = strings.TrimSpace(ecoURL)
	if ecoURL == "" {
		return ""
	}

	// Trim to slug after "openings/" or last slash.
	if idx := strings.LastIndex(ecoURL, "openings/"); idx != -1 {
		ecoURL = ecoURL[idx+len("openings/"):]
	} else if idx := strings.LastIndex(ecoURL, "/"); idx != -1 {
		ecoURL = ecoURL[idx+1:]
	}

	// Drop query params if any.
	if idx := strings.Index(ecoURL, "?"); idx != -1 {
		ecoURL = ecoURL[:idx]
	}

	// Remove move sequence suffix starting at first "-<digit>"
	if loc := reEcoMoves.FindStringIndex(ecoURL); loc != nil {
		ecoURL = ecoURL[:loc[0]]
	}

	// Replace separators and collapse whitespace.
	ecoURL = strings.ReplaceAll(ecoURL, "...", " ")
	ecoURL = strings.ReplaceAll(ecoURL, "-", " ")
	ecoURL = reSpaces.ReplaceAllString(ecoURL, " ")

	// Drop any trailing tokens that look like move numbers (e.g., "7.h3", "5...Bb6")
	fields := strings.Fields(ecoURL)
	for i, tok := range fields {
		if strings.Contains(tok, "...") {
			fields = fields[:i]
			break
		}
		if strings.IndexFunc(tok, unicode.IsDigit) != -1 {
			fields = fields[:i]
			break
		}
	}

	return strings.TrimSpace(strings.Join(fields, " "))
}

// converts string to int safely
func parsePositiveInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// BuildTagSummary maps the raw tag map into a typed summary and optionally
// computes POV info for povUser (case-insensitive). If povUser == "" it skips POV.
func BuildTagSummary(tags map[string]string, povUser string) TagSummary {
	toInt := func(s string) int {
		if s == "" {
			return 0
		}
		n, _ := strconv.Atoi(s)
		return n
	}
	s := TagSummary{
		Event:       tags["Event"],
		Site:        tags["Site"],
		Date:        tags["Date"],
		Round:       tags["Round"],
		Result:      tags["Result"],
		White:       tags["White"],
		Black:       tags["Black"],
		WhiteElo:    toInt(tags["WhiteElo"]),
		BlackElo:    toInt(tags["BlackElo"]),
		TimeControl: tags["TimeControl"],
		Termination: tags["Termination"],
		Link:        tags["Link"],
		ECO:         tags["ECO"],
		ECOUrl:      tags["ECOUrl"],
		UTCDate:     tags["UTCDate"],
		UTCTime:     tags["UTCTime"],
		CurrentFEN:  tags["CurrentPosition"],
	}

	if povUser != "" {
		u := strings.ToLower(povUser)
		if strings.ToLower(s.White) == u {
			s.Color = "white"
			s.Opponent = s.Black
			s.OppRating = s.BlackElo
		} else if strings.ToLower(s.Black) == u {
			s.Color = "black"
			s.Opponent = s.White
			s.OppRating = s.WhiteElo
		}
	}
	return s
}

func GetUnixTimeStamp(date string, timeStamp string, timeZone string) int64 {
	// Combine date and time fields into one string
	dateStr := fmt.Sprintf("%s %s", date, timeStamp)

	// Parse the combined date/time in UTC
	loc, _ := time.LoadLocation(timeZone)
	t, err := time.ParseInLocation(layout, dateStr, loc)
	if err != nil {
		panic(err)
	}

	return t.Unix()
}

func GetWorkerCount() int {
	//default number of workers = number of cpus. Otherwise can be overwritten with WORKERS env var
	n := runtime.NumCPU()
	if v := os.Getenv("WORKERS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			n = parsed
		}
	}
	return n
}

func IsEven(number int) bool {
	return number%2 == 0
}

// NormalizeFEN strips move counters and keeps only the structural position:
// <pieces> <side> <castling> <en-passant>
func NormalizeFEN(fen string) string {
	parts := strings.Split(fen, " ")
	if len(parts) < 4 {
		// malformed FEN, return original
		return fen
	}

	pieces := parts[0]
	side := parts[1]
	castling := parts[2]
	ep := parts[3]

	// OPTIONAL: normalize empty castling field "-" to something consistent
	if castling == "" {
		castling = "-"
	}

	// OPTIONAL: you can normalize "no EP square" as "-"
	if ep == "" {
		ep = "-"
	}

	return pieces + " " + side + " " + castling + " " + ep
}
