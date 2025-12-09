package models

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
	GameId      int
	Moves       []Move
	ECO         string `json:"eco"`
}

type Move struct {
	Move       string
	PlayedBy   string
	MoveNumber int
	Ply        int
	Color      string
	FenBefore  FENEval
	FenAfter   FENEval
	Analysis   MoveAnalysis

	//Used for reporting bad fens
	URL      string
	Opponent string
	ECO      string
}
type FENEval struct {
	MoveNumber int      `json:"move_number"` // fullmove number from FEN
	SideToMove string   `json:"side_to_move"`
	FEN        string   `json:"fen"`
	Score      UCIScore `json:"score"`
}

type MoveAnalysis struct {
	CPChange      int
	Is_Suboptimal bool
	Is_Innacuracy bool
	Is_Mistake    bool
	Is_Blunder    bool
}

//FENs where you've made a bad move and how many times you've done it
type SuboptimalFen struct {
	NormalizedFenBefore string
	TimesSeen           int
	SuboptimalCount     int
	InaccuracyCount     int
	MistakeCount        int
	ErrorCount          int
	ErrorRate           float64
}

//Further details for each bad FEN by game, what move you made, what you should have made, etc
type SuboptimalFensReport struct {
	BadFen SuboptimalFen
	Moves  []Move
}
