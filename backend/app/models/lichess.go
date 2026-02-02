package models

// LichessGame represents a subset of the Lichess NDJSON game export payload.
type LichessGame struct {
	ID         string `json:"id"`
	CreatedAt  int64  `json:"createdAt"`
	LastMoveAt int64  `json:"lastMoveAt"`
	Speed      string `json:"speed"`
	Perf       string `json:"perf"`
	Rated      bool   `json:"rated"`
	PGN        string `json:"pgn"`
	Winner     string `json:"winner"`
	Clock      *struct {
		Initial   int `json:"initial"`
		Increment int `json:"increment"`
	} `json:"clock"`
	Opening *struct {
		ECO  string `json:"eco"`
		Name string `json:"name"`
	} `json:"opening"`
	Players struct {
		White LichessPlayer `json:"white"`
		Black LichessPlayer `json:"black"`
	} `json:"players"`
}

// LichessPlayer represents a player in the Lichess game payload.
type LichessPlayer struct {
	User *struct {
		Name string `json:"name"`
	} `json:"user"`
	Name   string `json:"name"`
	Rating int    `json:"rating"`
}
