package config

import (
	"log"
	"os"
	"strconv"

	// this will automatically load your .env file:
	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	Logs     LogConfig
	DB       PostgresConfig
	Engine   EngineConfig
	QueueURL string
}

type LogConfig struct {
	Style string
	Level string
}
type PostgresConfig struct {
	Username string
	Password string
	URL      string
	Port     string
}

type EngineConfig struct {
	Path        string
	MoveTime    int
	DepthOrTime bool //true for depth, false for time
	Depth       int
	NumMoves    int //how many moves should the engine process
	NumGames    int
}

func LoadConfig() (*Config, error) {
	moveTime, err := strconv.Atoi(os.Getenv("ENGINE_MOVE_TIME"))
	if err != nil {
		log.Fatalf("Error converting string to int: ENGINE_MOVE_TIME: %v", err)
	}

	numMoves, err := strconv.Atoi(os.Getenv("ENGINE_NUMBER_OF_MOVES"))
	if err != nil {
		log.Fatalf("Error converting string to int: ENGINE_NUMBER_OF_MOVES: %v", err)
	}

	numGames, err := strconv.Atoi(os.Getenv("ENGINE_NUMBER_OF_GAMES"))
	if err != nil {
		log.Fatalf("Error converting string to int: ENGINE_NUMBER_OF_GAMES: %v", err)
	}

	depth, err := strconv.Atoi(os.Getenv("ENGINE_DEPTH"))
	if err != nil {
		log.Fatalf("Error converting string to int: ENGINE_DEPTH_OR_TIME: %v", err)
	}

	depthOrTime, err := strconv.ParseBool(os.Getenv("ENGINE_DEPTH_OR_TIME"))
	if err != nil {
		log.Fatalf("Error parsing ENGINE_DEPTH_OR_TIME: %v", err)
	}

	cfg := &Config{
		QueueURL: os.Getenv("QUEUE_URL"),
		Logs: LogConfig{
			Style: os.Getenv("LOG_STYLE"),
			Level: os.Getenv("LOG_LEVEL"),
		},
		DB: PostgresConfig{
			Username: os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PWD"),
			URL:      os.Getenv("POSTGRES_URL"),
			Port:     os.Getenv("POSTGRES_PORT"),
		},
		Engine: EngineConfig{
			Path:        os.Getenv("ENGINE_PATH"),
			MoveTime:    moveTime,
			Depth:       depth,
			DepthOrTime: depthOrTime,
			NumMoves:    numMoves,
			NumGames:    numGames,
		},
	}

	return cfg, nil
}
