package config

import "testing"

func TestLoadUsesMockLLMWhenGeminiKeyIsMissing(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "interview_ai")
	t.Setenv("POSTGRES_PASSWORD", "password")
	t.Setenv("POSTGRES_DB", "interview_ai")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_MODEL", "")
	t.Setenv("GEMINI_FALLBACK_MODEL", "")

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
}

func TestLoadTreatsWhitespaceGeminiKeyAsMissing(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "interview_ai")
	t.Setenv("POSTGRES_PASSWORD", "password")
	t.Setenv("POSTGRES_DB", "interview_ai")
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
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("GEMINI_MODEL", "gemini-custom-primary")
	t.Setenv("GEMINI_FALLBACK_MODEL", "gemini-custom-fallback")

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
}
