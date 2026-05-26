package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"interview-ai/backend/internal/config"
	"interview-ai/backend/internal/llm"
)

func TestHealthReturnsOK(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	newRouter(nil).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON response, got error: %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
}

func TestQuestionGeneratorForConfigUsesMockWithoutGeminiKey(t *testing.T) {
	generator := questionGeneratorForConfig(config.Config{
		GeminiAPIKey:        "",
		GeminiModel:         "gemini-2.5-flash",
		GeminiFallbackModel: "gemini-2.5-flash-lite",
	})

	if _, ok := generator.(llm.MockQuestionGenerator); !ok {
		t.Fatalf("expected MockQuestionGenerator, got %T", generator)
	}
}

func TestQuestionGeneratorForConfigUsesGeminiWithGeminiKey(t *testing.T) {
	generator := questionGeneratorForConfig(config.Config{
		GeminiAPIKey:        "test-key",
		GeminiModel:         "gemini-2.5-flash",
		GeminiFallbackModel: "gemini-2.5-flash-lite",
	})

	if _, ok := generator.(*llm.GeminiQuestionGenerator); !ok {
		t.Fatalf("expected GeminiQuestionGenerator, got %T", generator)
	}
}
