package config

import "testing"

func TestLoadUsesMockLLMWhenGeminiKeyIsMissing(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "interview_ai")
	t.Setenv("POSTGRES_PASSWORD", "password")
	t.Setenv("POSTGRES_DB", "interview_ai")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_MODEL", "")
	t.Setenv("GEMINI_FALLBACK_MODEL", "")
	t.Setenv("GEMINI_ANSWER_MODEL", "")
	t.Setenv("GEMINI_ANSWER_FALLBACK_MODEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.GeminiAPIKey != "" {
		t.Fatalf("expected empty GeminiAPIKey, got %q", cfg.GeminiAPIKey)
	}
	if cfg.GeminiModel != "gemini-2.5-flash" {
		t.Fatalf("expected default model gemini-2.5-flash, got %q", cfg.GeminiModel)
	}
	if cfg.GeminiFallbackModel != "gemini-2.5-flash-lite" {
		t.Fatalf("expected default fallback model gemini-2.5-flash-lite, got %q", cfg.GeminiFallbackModel)
	}
	if cfg.GeminiAnswerModel != cfg.GeminiModel {
		t.Fatalf("expected answer model to default to question model, got %q", cfg.GeminiAnswerModel)
	}
	if cfg.GeminiAnswerFallbackModel != cfg.GeminiFallbackModel {
		t.Fatalf("expected answer fallback model to default to question fallback model, got %q", cfg.GeminiAnswerFallbackModel)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("expected default log level info, got %q", cfg.LogLevel)
	}
}

func TestLoadTreatsWhitespaceGeminiKeyAsMissing(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "interview_ai")
	t.Setenv("POSTGRES_PASSWORD", "password")
	t.Setenv("POSTGRES_DB", "interview_ai")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("GEMINI_API_KEY", "   ")
	t.Setenv("GEMINI_MODEL", "")
	t.Setenv("GEMINI_FALLBACK_MODEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.GeminiAPIKey != "" {
		t.Fatalf("expected whitespace GeminiAPIKey to be empty, got %q", cfg.GeminiAPIKey)
	}
}

func TestLoadReadsGeminiConfig(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "interview_ai")
	t.Setenv("POSTGRES_PASSWORD", "password")
	t.Setenv("POSTGRES_DB", "interview_ai")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("GEMINI_MODEL", "gemini-custom-primary")
	t.Setenv("GEMINI_FALLBACK_MODEL", "gemini-custom-fallback")
	t.Setenv("GEMINI_ANSWER_MODEL", "gemini-answer-primary")
	t.Setenv("GEMINI_ANSWER_FALLBACK_MODEL", "gemini-answer-fallback")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.GeminiAPIKey != "test-key" {
		t.Fatalf("expected GeminiAPIKey test-key, got %q", cfg.GeminiAPIKey)
	}
	if cfg.GeminiModel != "gemini-custom-primary" {
		t.Fatalf("expected GeminiModel gemini-custom-primary, got %q", cfg.GeminiModel)
	}
	if cfg.GeminiFallbackModel != "gemini-custom-fallback" {
		t.Fatalf("expected GeminiFallbackModel gemini-custom-fallback, got %q", cfg.GeminiFallbackModel)
	}
	if cfg.GeminiAnswerModel != "gemini-answer-primary" {
		t.Fatalf("expected GeminiAnswerModel gemini-answer-primary, got %q", cfg.GeminiAnswerModel)
	}
	if cfg.GeminiAnswerFallbackModel != "gemini-answer-fallback" {
		t.Fatalf("expected GeminiAnswerFallbackModel gemini-answer-fallback, got %q", cfg.GeminiAnswerFallbackModel)
	}
}

func TestLoadReadsLogLevel(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "interview_ai")
	t.Setenv("POSTGRES_PASSWORD", "password")
	t.Setenv("POSTGRES_DB", "interview_ai")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_MODEL", "")
	t.Setenv("GEMINI_FALLBACK_MODEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.LogLevel != "debug" {
		t.Fatalf("expected log level debug, got %q", cfg.LogLevel)
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "interview_ai")
	t.Setenv("POSTGRES_PASSWORD", "password")
	t.Setenv("POSTGRES_DB", "interview_ai")
	t.Setenv("LOG_LEVEL", "trace")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_MODEL", "")
	t.Setenv("GEMINI_FALLBACK_MODEL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected Load to reject invalid LOG_LEVEL")
	}
}
