package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type AudioStorage interface {
	SaveAnswerAudio(ctx context.Context, interviewID string, questionID string, file io.Reader) (string, error)
	DeleteAnswerAudio(ctx context.Context, audioPath string) error
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

func (s *LocalAudioStorage) DeleteAnswerAudio(ctx context.Context, audioPath string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	filePath, err := s.localPathForStoredAudioPath(audioPath)
	if err != nil {
		return err
	}

	if err := os.Remove(filePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}

func (s *LocalAudioStorage) localPathForStoredAudioPath(audioPath string) (string, error) {
	cleanPath := filepath.Clean(audioPath)
	audioPrefix := filepath.Join("storage", "audio")
	if cleanPath == audioPrefix || !strings.HasPrefix(cleanPath, audioPrefix+string(filepath.Separator)) {
		return "", fmt.Errorf("audio path must be under %s", audioPrefix)
	}

	relativePath, err := filepath.Rel(audioPrefix, cleanPath)
	if err != nil {
		return "", err
	}
	if relativePath == "." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) || relativePath == ".." {
		return "", fmt.Errorf("audio path must be under %s", audioPrefix)
	}

	rootAbs, err := filepath.Abs(s.root)
	if err != nil {
		return "", err
	}
	filePath := filepath.Join(rootAbs, relativePath)
	fileAbs, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}
	if fileAbs != rootAbs && !strings.HasPrefix(fileAbs, rootAbs+string(filepath.Separator)) {
		return "", fmt.Errorf("audio path escapes storage root")
	}

	return fileAbs, nil
}
