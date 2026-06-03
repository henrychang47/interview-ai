package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHealthReturnsOK(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	NewRouter(nil, t.TempDir()).ServeHTTP(response, request)

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

func TestRouterLogsHealthRequest(t *testing.T) {
	var logs bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	NewRouter(nil, t.TempDir()).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	logOutput := logs.String()
	for _, expected := range []string{
		`"method":"GET"`,
		`"path":"/health"`,
		`"status":200`,
		`"duration_ms":`,
	} {
		if !strings.Contains(logOutput, expected) {
			t.Fatalf("expected request log to contain %s, got %s", expected, logOutput)
		}
	}
}

func TestAudioRouteServesUploadedAudio(t *testing.T) {
	tempDir := t.TempDir()
	audioDir := filepath.Join(tempDir, "audio")
	if err := os.MkdirAll(filepath.Join(audioDir, "interview-id"), 0o755); err != nil {
		t.Fatalf("create audio dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(audioDir, "interview-id", "question-id.webm"), []byte("webm-bytes"), 0o644); err != nil {
		t.Fatalf("write audio file: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/audio/interview-id/question-id.webm", nil)
	response := httptest.NewRecorder()

	NewRouter(nil, audioDir).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if response.Body.String() != "webm-bytes" {
		t.Fatalf("expected audio bytes, got %q", response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "audio/webm") {
		t.Fatalf("expected audio/webm content type, got %q", contentType)
	}
}

func TestAudioRouteReturnsNotFoundForMissingAudio(t *testing.T) {
	audioDir := t.TempDir()
	request := httptest.NewRequest(http.MethodGet, "/audio/interview-id/missing.webm", nil)
	response := httptest.NewRecorder()

	NewRouter(nil, audioDir).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
}
