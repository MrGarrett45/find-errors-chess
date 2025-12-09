package models

type player struct {
	Username string `json:"username"`
	Result   string `json:"result"`
	Rating   int    `json:"rating"`
}

// Model received from chess.com
type Game struct {
	URL         string `json:"url"`
	PGN         string `json:"pgn"`
	TimeControl string `json:"time_control"`
	TimeClass   string `json:"time_class"`
	Rated       bool   `json:"rated"`
	EndTime     int64  `json:"end_time"`
	Rules       string `json:"rules"`
	White       player `json:"white"`
	Black       player `json:"black"`
	ECO         string `json:"eco"`
	// Some months also include ECO/opening inside PGN tags only; you can parse later.
}
