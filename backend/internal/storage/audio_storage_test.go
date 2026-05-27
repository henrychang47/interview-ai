package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalAudioStorageSavesAnswerAudio(t *testing.T) {
	root := t.TempDir()
	storage := NewLocalAudioStorage(root)

	audioPath, err := storage.SaveAnswerAudio(
		context.Background(),
		"interview-id",
		"question-id",
		strings.NewReader("webm-bytes"),
	)

	if err != nil {
		t.Fatalf("SaveAnswerAudio returned error: %v", err)
	}
	expectedPath := filepath.Join("storage", "audio", "interview-id", "question-id.webm")
	if audioPath != expectedPath {
		t.Fatalf("expected audio path %q, got %q", expectedPath, audioPath)
	}

	savedBytes, err := os.ReadFile(filepath.Join(root, "interview-id", "question-id.webm"))
	if err != nil {
		t.Fatalf("read saved audio: %v", err)
	}
	if string(savedBytes) != "webm-bytes" {
		t.Fatalf("expected saved bytes, got %q", string(savedBytes))
	}
}
