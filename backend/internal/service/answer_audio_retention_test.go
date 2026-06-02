package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"interview-ai/backend/internal/model"
)

func TestCleanupExpiredAnswerAudioDeletesFilesAndClearsAudioPaths(t *testing.T) {
	cutoff := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	repository := &stubAnswerAudioRetentionRepository{
		expired: []model.ExpiredAnswerAudio{
			{AnswerID: "answer-1", AudioPath: "storage/audio/interview-1/question-1.webm"},
			{AnswerID: "answer-2", AudioPath: "storage/audio/interview-1/question-2.webm"},
		},
	}
	storage := &stubAnswerAudioRetentionStorage{}
	retention := NewAnswerAudioRetentionService(repository, storage)

	result, err := retention.CleanupExpiredAnswerAudio(context.Background(), cutoff)

	if err != nil {
		t.Fatalf("CleanupExpiredAnswerAudio returned error: %v", err)
	}
	if repository.cutoff != cutoff {
		t.Fatalf("expected cutoff %v, got %v", cutoff, repository.cutoff)
	}
	if result.Deleted != 2 {
		t.Fatalf("expected 2 deleted files, got %d", result.Deleted)
	}
	if len(storage.deletedPaths) != 2 {
		t.Fatalf("expected 2 deleted paths, got %d", len(storage.deletedPaths))
	}
	if len(repository.cleared) != 2 {
		t.Fatalf("expected 2 cleared answers, got %d", len(repository.cleared))
	}
	if repository.cleared[0] != "answer-1|storage/audio/interview-1/question-1.webm" {
		t.Fatalf("expected first cleared answer/path pair, got %q", repository.cleared[0])
	}
	if repository.cleared[1] != "answer-2|storage/audio/interview-1/question-2.webm" {
		t.Fatalf("expected second cleared answer/path pair, got %q", repository.cleared[1])
	}
}

func TestCleanupExpiredAnswerAudioDoesNotClearPathWhenDeleteFails(t *testing.T) {
	deleteErr := errors.New("permission denied")
	repository := &stubAnswerAudioRetentionRepository{
		expired: []model.ExpiredAnswerAudio{
			{AnswerID: "answer-1", AudioPath: "storage/audio/interview-1/question-1.webm"},
		},
	}
	storage := &stubAnswerAudioRetentionStorage{err: deleteErr}
	retention := NewAnswerAudioRetentionService(repository, storage)

	result, err := retention.CleanupExpiredAnswerAudio(context.Background(), time.Now())

	if !errors.Is(err, deleteErr) {
		t.Fatalf("expected delete error, got %v", err)
	}
	if result.Deleted != 0 {
		t.Fatalf("expected 0 deleted files, got %d", result.Deleted)
	}
	if len(repository.cleared) != 0 {
		t.Fatalf("expected audio path not to be cleared, got %+v", repository.cleared)
	}
}

type stubAnswerAudioRetentionRepository struct {
	cutoff  time.Time
	expired []model.ExpiredAnswerAudio
	cleared []string
	err     error
}

func (r *stubAnswerAudioRetentionRepository) ListExpiredAnswerAudio(ctx context.Context, cutoff time.Time) ([]model.ExpiredAnswerAudio, error) {
	r.cutoff = cutoff
	if r.err != nil {
		return nil, r.err
	}
	return r.expired, nil
}

func (r *stubAnswerAudioRetentionRepository) ClearAnswerAudioPath(ctx context.Context, answerID string, audioPath string) error {
	r.cleared = append(r.cleared, answerID+"|"+audioPath)
	return r.err
}

type stubAnswerAudioRetentionStorage struct {
	deletedPaths []string
	err          error
}

func (s *stubAnswerAudioRetentionStorage) DeleteAnswerAudio(ctx context.Context, audioPath string) error {
	s.deletedPaths = append(s.deletedPaths, audioPath)
	return s.err
}
