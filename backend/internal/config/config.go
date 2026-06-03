package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const DevelopmentIPHashSalt = "development-ip-hash-salt"

type Config struct {
	DatabaseURL                  string
	LogLevel                     string
	InterviewCreationLimitPer24H int
	IPHashSalt                   string
	GeminiAPIKey                 string
	GeminiModel                  string
	GeminiFallbackModel          string
	GeminiAnswerModel            string
	GeminiAnswerFallbackModel    string
	GeminiTTSModel               string
	GeminiTTSFallbackModel       string
	GeminiTTSVoice               string
}

func Load() (Config, error) {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if appEnv == "" {
		appEnv = "development"
	}
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresPort := os.Getenv("POSTGRES_PORT")
	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresDB := os.Getenv("POSTGRES_DB")
	logLevel := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	if logLevel == "" {
		logLevel = "info"
	}
	interviewCreationLimitPer24H := 5
	interviewCreationLimitValue := strings.TrimSpace(os.Getenv("INTERVIEW_CREATION_LIMIT_PER_24H"))
	if interviewCreationLimitValue != "" {
		parsedLimit, err := strconv.Atoi(interviewCreationLimitValue)
		if err != nil {
			return Config{}, errors.New("INTERVIEW_CREATION_LIMIT_PER_24H must be a positive integer")
		}
		interviewCreationLimitPer24H = parsedLimit
	}
	ipHashSalt := strings.TrimSpace(os.Getenv("IP_HASH_SALT"))
	if ipHashSalt == "" && appEnv == "development" {
		ipHashSalt = DevelopmentIPHashSalt
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
	geminiAnswerModel := strings.TrimSpace(os.Getenv("GEMINI_ANSWER_MODEL"))
	if geminiAnswerModel == "" {
		geminiAnswerModel = geminiModel
	}
	geminiAnswerFallbackModel := strings.TrimSpace(os.Getenv("GEMINI_ANSWER_FALLBACK_MODEL"))
	if geminiAnswerFallbackModel == "" {
		geminiAnswerFallbackModel = geminiFallbackModel
	}
	geminiTTSModel := strings.TrimSpace(os.Getenv("GEMINI_TTS_MODEL"))
	if geminiTTSModel == "" {
		geminiTTSModel = "gemini-3.1-flash-tts-preview"
	}
	geminiTTSFallbackModel := strings.TrimSpace(os.Getenv("GEMINI_TTS_FALLBACK_MODEL"))
	if geminiTTSFallbackModel == "" {
		geminiTTSFallbackModel = "gemini-2.5-flash-preview-tts"
	}
	geminiTTSVoice := strings.TrimSpace(os.Getenv("GEMINI_TTS_VOICE"))
	if geminiTTSVoice == "" {
		geminiTTSVoice = "Kore"
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
	if interviewCreationLimitPer24H <= 0 {
		return Config{}, errors.New("INTERVIEW_CREATION_LIMIT_PER_24H must be a positive integer")
	}
	if ipHashSalt == "" {
		return Config{}, errors.New("IP_HASH_SALT is required outside development")
	}
	if appEnv != "development" && ipHashSalt == DevelopmentIPHashSalt {
		return Config{}, errors.New("IP_HASH_SALT must not use the development placeholder outside development")
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
		DatabaseURL:                  databaseURL,
		LogLevel:                     logLevel,
		InterviewCreationLimitPer24H: interviewCreationLimitPer24H,
		IPHashSalt:                   ipHashSalt,
		GeminiAPIKey:                 geminiAPIKey,
		GeminiModel:                  geminiModel,
		GeminiFallbackModel:          geminiFallbackModel,
		GeminiAnswerModel:            geminiAnswerModel,
		GeminiAnswerFallbackModel:    geminiAnswerFallbackModel,
		GeminiTTSModel:               geminiTTSModel,
		GeminiTTSFallbackModel:       geminiTTSFallbackModel,
		GeminiTTSVoice:               geminiTTSVoice,
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
