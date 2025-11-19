package config

import (
	"log"
	"os"
	"strconv"

	// this will automatically load your .env file:
	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	Logs   LogConfig
	DB     PostgresConfig
	User   string
	Engine EngineConfig
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
}

func LoadConfig() (*Config, error) {
	moveTime, err := strconv.Atoi(os.Getenv("ENGINE_MOVE_TIME"))
	if err != nil {
		log.Fatalf("Error converting string to int: %v", err)
	}

	depth, err := strconv.Atoi(os.Getenv("ENGINE_DEPTH"))
	if err != nil {
		log.Fatalf("Error parsing DEPTH_OR_TIME: %v", err)
	}

	depthOrTime, err := strconv.ParseBool(os.Getenv("DEPTH_OR_TIME"))
	if err != nil {
		log.Fatalf("Error parsing DEPTH_OR_TIME: %v", err)
	}

	cfg := &Config{
		User: os.Getenv("USER"),
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
		},
	}

	return cfg, nil
}
