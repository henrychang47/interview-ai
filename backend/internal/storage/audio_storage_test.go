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

func TestLocalAudioStorageDeletesAnswerAudio(t *testing.T) {
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

	if err := storage.DeleteAnswerAudio(context.Background(), audioPath); err != nil {
		t.Fatalf("DeleteAnswerAudio returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "interview-id", "question-id.webm")); !os.IsNotExist(err) {
		t.Fatalf("expected deleted audio file, got err %v", err)
	}
}

func TestLocalAudioStorageDeleteAnswerAudioTreatsMissingFileAsDeleted(t *testing.T) {
	storage := NewLocalAudioStorage(t.TempDir())

	err := storage.DeleteAnswerAudio(context.Background(), filepath.Join("storage", "audio", "interview-id", "missing.webm"))

	if err != nil {
		t.Fatalf("expected missing audio delete to succeed, got %v", err)
	}
}

func TestLocalAudioStorageDeleteAnswerAudioRejectsPathOutsideAudioRoot(t *testing.T) {
	storage := NewLocalAudioStorage(t.TempDir())

	err := storage.DeleteAnswerAudio(context.Background(), filepath.Join("storage", "audio", "..", "secret.webm"))

	if err == nil {
		t.Fatal("expected path outside audio root to be rejected")
	}
}
