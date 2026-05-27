package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type AudioStorage interface {
	SaveAnswerAudio(ctx context.Context, interviewID string, questionID string, file io.Reader) (string, error)
}

type LocalAudioStorage struct {
	root string
}

func NewLocalAudioStorage(root string) *LocalAudioStorage {
	return &LocalAudioStorage{root: root}
}

func (s *LocalAudioStorage) SaveAnswerAudio(ctx context.Context, interviewID string, questionID string, file io.Reader) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	interviewDir := filepath.Join(s.root, interviewID)
	if err := os.MkdirAll(interviewDir, 0o755); err != nil {
		return "", err
	}

	filePath := filepath.Join(interviewDir, questionID+".webm")
	destination, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer destination.Close()

	if _, err := io.Copy(destination, file); err != nil {
		return "", err
	}

	return filepath.Join("storage", "audio", interviewID, questionID+".webm"), nil
}
