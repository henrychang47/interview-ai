package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL         string
	GeminiAPIKey        string
	GeminiModel         string
	GeminiFallbackModel string
}

func Load() (Config, error) {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresPort := os.Getenv("POSTGRES_PORT")
	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresDB := os.Getenv("POSTGRES_DB")
	geminiAPIKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	geminiModel := strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	if geminiModel == "" {
		geminiModel = "gemini-2.5-flash"
	}
	geminiFallbackModel := strings.TrimSpace(os.Getenv("GEMINI_FALLBACK_MODEL"))
	if geminiFallbackModel == "" {
		geminiFallbackModel = "gemini-2.5-flash-lite"
	}

	if postgresHost == "" {
		return Config{}, errors.New("POSTGRES_HOST is required")
	}
	if postgresPort == "" {
		return Config{}, errors.New("POSTGRES_PORT is required")
	}
	if postgresUser == "" {
		return Config{}, errors.New("POSTGRES_USER is required")
	}
	if postgresPassword == "" {
		return Config{}, errors.New("POSTGRES_PASSWORD is required")
	}
	if postgresDB == "" {
		return Config{}, errors.New("POSTGRES_DB is required")
	}

	databaseURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		postgresUser,
		postgresPassword,
		postgresHost,
		postgresPort,
		postgresDB,
	)

	return Config{
		DatabaseURL:         databaseURL,
		GeminiAPIKey:        geminiAPIKey,
		GeminiModel:         geminiModel,
		GeminiFallbackModel: geminiFallbackModel,
	}, nil
}
