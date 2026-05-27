package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	"interview-ai/backend/internal/model"
)

var (
	ErrAudioFileRequired     = errors.New("audio file is required")
	ErrUnsupportedAudioType  = errors.New("audio file must be audio/webm")
	ErrSaveAnswerAudioFailed = errors.New("failed to save answer audio")
)

type AudioStorage interface {
	SaveAnswerAudio(ctx context.Context, interviewID string, questionID string, file io.Reader) (string, error)
}

type AnswerRepository interface {
	EnsureQuestionForInterview(ctx context.Context, interviewID string, questionID string) error
	UpsertAnswer(ctx context.Context, interviewID string, questionID string, audioPath string) (model.Answer, error)
}

type UploadAnswerInput struct {
	InterviewID string
	QuestionID  string
	ContentType string
	Audio       io.Reader
}

type AnswerService struct {
	storage    AudioStorage
	repository AnswerRepository
}

func NewAnswerService(storage AudioStorage, repository AnswerRepository) *AnswerService {
	return &AnswerService{storage: storage, repository: repository}
}

func (s *AnswerService) UploadAnswer(ctx context.Context, input UploadAnswerInput) (model.UploadAnswerResponse, error) {
	if input.Audio == nil {
		return model.UploadAnswerResponse{}, ErrAudioFileRequired
	}
	if input.ContentType != "audio/webm" {
		return model.UploadAnswerResponse{}, ErrUnsupportedAudioType
	}

	if err := s.repository.EnsureQuestionForInterview(ctx, input.InterviewID, input.QuestionID); err != nil {
		return model.UploadAnswerResponse{}, err
	}

	audioPath, err := s.storage.SaveAnswerAudio(ctx, input.InterviewID, input.QuestionID, input.Audio)
	if err != nil {
		return model.UploadAnswerResponse{}, fmt.Errorf("%w: %v", ErrSaveAnswerAudioFailed, err)
	}

	answer, err := s.repository.UpsertAnswer(ctx, input.InterviewID, input.QuestionID, audioPath)
	if err != nil {
		return model.UploadAnswerResponse{}, err
	}

	response := model.UploadAnswerResponse{
		ID:             answer.ID,
		InterviewID:    answer.InterviewID,
		QuestionID:     answer.QuestionID,
		TranscriptText: answer.TranscriptText,
	}
	if answer.AudioPath != nil {
		response.AudioPath = *answer.AudioPath
	}

	return response, nil
}
