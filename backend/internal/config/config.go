package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
}

func Load() (Config, error) {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresPort := os.Getenv("POSTGRES_PORT")
	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresDB := os.Getenv("POSTGRES_DB")

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

	return Config{DatabaseURL: databaseURL}, nil
}
