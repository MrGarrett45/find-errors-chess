package models

type PositionEval struct {
	MoveNumber int      `json:"move_number"` // fullmove number from FEN
	SideToMove string   `json:"side_to_move"`
	FEN        string   `json:"fen"`
	Score      UCIScore `json:"score"`
	Move       string
}

type GameEval struct {
	URL      string         `json:"url"`
	When     int64          `json:"when_unix"`
	Color    string         `json:"color"`
	Opponent string         `json:"opponent"`
	Result   string         `json:"result"`
	Evals    []PositionEval `json:"evals"`
	Summary  UCIScore       `json:"summary_final_position"`
}

// What we return to the frontend and store in DB (trimmed & consistent DTO)
type GameLite struct {
	URL         string `json:"url"`
	When        int64  `json:"when_unix"`
	Color       string `json:"color"` // "white" or "black"
	Opponent    string `json:"opponent"`
	OppRating   int    `json:"opponent_rating"`
	Result      string `json:"result"` // "win","checkmated","resigned", etc. (as Chess.com reports)
	Rated       bool   `json:"rated"`
	TimeClass   string `json:"time_class"`   // blitz/rapid/bullet/daily
	TimeControl string `json:"time_control"` // e.g. "600+0"
	PGN         string `json:"pgn"`          // included for convenience (you can omit if payload too big)
}
