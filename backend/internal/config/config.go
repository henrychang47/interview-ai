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
	LogLevel            string
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
	logLevel := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	if logLevel == "" {
		logLevel = "info"
	}
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
	if !isSupportedLogLevel(logLevel) {
		return Config{}, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error")
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
		LogLevel:            logLevel,
		GeminiAPIKey:        geminiAPIKey,
		GeminiModel:         geminiModel,
		GeminiFallbackModel: geminiFallbackModel,
	}, nil
}

func isSupportedLogLevel(level string) bool {
	switch level {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}
