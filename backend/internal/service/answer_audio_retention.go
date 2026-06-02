package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"interview-ai/backend/internal/model"
)

type AnswerAudioRetentionRepository interface {
	ListExpiredAnswerAudio(ctx context.Context, cutoff time.Time) ([]model.ExpiredAnswerAudio, error)
	ClearAnswerAudioPath(ctx context.Context, answerID string, audioPath string) error
}

type AnswerAudioRetentionStorage interface {
	DeleteAnswerAudio(ctx context.Context, audioPath string) error
}

type AnswerAudioCleanupResult struct {
	Scanned int
	Deleted int
	Failed  int
}

type AnswerAudioRetentionService struct {
	repository AnswerAudioRetentionRepository
	storage    AnswerAudioRetentionStorage
}

func NewAnswerAudioRetentionService(repository AnswerAudioRetentionRepository, storage AnswerAudioRetentionStorage) *AnswerAudioRetentionService {
	return &AnswerAudioRetentionService{repository: repository, storage: storage}
}

func (s *AnswerAudioRetentionService) CleanupExpiredAnswerAudio(ctx context.Context, cutoff time.Time) (AnswerAudioCleanupResult, error) {
	if s == nil || s.repository == nil || s.storage == nil {
		return AnswerAudioCleanupResult{}, nil
	}

	expiredAudio, err := s.repository.ListExpiredAnswerAudio(ctx, cutoff)
	if err != nil {
		return AnswerAudioCleanupResult{}, err
	}

	result := AnswerAudioCleanupResult{Scanned: len(expiredAudio)}
	var cleanupErr error
	for _, audio := range expiredAudio {
		if err := s.storage.DeleteAnswerAudio(ctx, audio.AudioPath); err != nil {
			result.Failed++
			cleanupErr = errors.Join(cleanupErr, err)
			slog.Error("delete expired answer audio", "answer_id", audio.AnswerID, "audio_path", audio.AudioPath, "error", err)
			continue
		}
		if err := s.repository.ClearAnswerAudioPath(ctx, audio.AnswerID, audio.AudioPath); err != nil {
			result.Failed++
			cleanupErr = errors.Join(cleanupErr, err)
			slog.Error("clear expired answer audio path", "answer_id", audio.AnswerID, "audio_path", audio.AudioPath, "error", err)
			continue
		}
		result.Deleted++
	}

	return result, cleanupErr
}

func StartAnswerAudioRetentionJob(ctx context.Context, retention *AnswerAudioRetentionService, retentionPeriod time.Duration, interval time.Duration) {
	if retention == nil || retentionPeriod <= 0 || interval <= 0 {
		return
	}

	runCleanup := func() {
		cutoff := time.Now().Add(-retentionPeriod)
		result, err := retention.CleanupExpiredAnswerAudio(context.Background(), cutoff)
		if err != nil {
			slog.Error("answer audio retention cleanup completed with errors", "scanned", result.Scanned, "deleted", result.Deleted, "failed", result.Failed, "error", err)
			return
		}
		slog.Info("answer audio retention cleanup completed", "scanned", result.Scanned, "deleted", result.Deleted, "failed", result.Failed)
	}

	go func() {
		runCleanup()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runCleanup()
			}
		}
	}()
}
