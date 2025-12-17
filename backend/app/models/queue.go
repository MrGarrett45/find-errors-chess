package models

type JobMessage struct {
	User       string `json:"user"`
	BatchIndex int    `json:"batch_index"` // 0-based
	NumGames   int    `json:"num_games"`   // usually 100
	JobID      string `json:"job_id"`      // optional, for progress tracking
	EngineDepth       int  `json:"engine_depth"`
	EngineMoveTime    int  `json:"engine_move_time"`
	EngineUseDepth    bool `json:"engine_use_depth"`
}
