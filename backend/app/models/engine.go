package models

type UCIScore struct {
	// Exactly one of these will be set:
	CP   *int   `json:"cp,omitempty"`   // centipawns, positive means advantage for side to move
	Mate *int   `json:"mate,omitempty"` // in N, sign indicates who is mating (+ means side to move mates)
	Best string `json:"bestmove"`       // engine best move in UCI, e.g. "e2e4"
}

// EngineSettings drives how we query Stockfish for a position.
type EngineSettings struct {
	Depth       int  `json:"depth"`
	MoveTimeMS  int  `json:"move_time_ms"`
	UseDepth    bool `json:"use_depth"` // if false, use movetime
}
