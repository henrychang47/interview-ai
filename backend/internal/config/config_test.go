package config

import "testing"

func TestLoadUsesMockLLMWhenOpenAIKeyIsMissing(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "interview_ai")
	t.Setenv("POSTGRES_PASSWORD", "password")
	t.Setenv("POSTGRES_DB", "interview_ai")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.OpenAIAPIKey != "" {
		t.Fatalf("expected empty OpenAIAPIKey, got %q", cfg.OpenAIAPIKey)
	}
	if cfg.OpenAIModel != "gpt-5.4-mini" {
		t.Fatalf("expected default model gpt-5.4-mini, got %q", cfg.OpenAIModel)
	}
}

func TestLoadTreatsWhitespaceOpenAIKeyAsMissing(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "interview_ai")
	t.Setenv("POSTGRES_PASSWORD", "password")
	t.Setenv("POSTGRES_DB", "interview_ai")
	t.Setenv("OPENAI_API_KEY", "   ")
	t.Setenv("OPENAI_MODEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.OpenAIAPIKey != "" {
		t.Fatalf("expected whitespace OpenAIAPIKey to be empty, got %q", cfg.OpenAIAPIKey)
	}
}

func TestLoadReadsOpenAIConfig(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "interview_ai")
	t.Setenv("POSTGRES_PASSWORD", "password")
	t.Setenv("POSTGRES_DB", "interview_ai")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_MODEL", "gpt-5.4")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.OpenAIAPIKey != "test-key" {
		t.Fatalf("expected OpenAIAPIKey test-key, got %q", cfg.OpenAIAPIKey)
	}
	if cfg.OpenAIModel != "gpt-5.4" {
		t.Fatalf("expected OpenAIModel gpt-5.4, got %q", cfg.OpenAIModel)
	}
}
