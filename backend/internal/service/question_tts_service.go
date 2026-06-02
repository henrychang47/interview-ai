package service

import (
	"context"
	"errors"
	"fmt"

	"interview-ai/backend/internal/model"

	"github.com/jackc/pgx/v5"
)

var ErrQuestionTTSUnavailable = errors.New("question TTS is unavailable")

type QuestionTTSRepository interface {
	GetByID(ctx context.Context, interviewID string) (model.InterviewDetail, error)
}

type QuestionTTSGenerator interface {
	GenerateQuestionSpeech(ctx context.Context, input model.QuestionTTSInput) ([]byte, error)
}

type QuestionTTSService struct {
	repository QuestionTTSRepository
	generator  QuestionTTSGenerator
}

func NewQuestionTTSService(repository QuestionTTSRepository, generator QuestionTTSGenerator) *QuestionTTSService {
	return &QuestionTTSService{repository: repository, generator: generator}
}

func (s *QuestionTTSService) GenerateQuestionSpeech(ctx context.Context, interviewID string, questionID string) ([]byte, error) {
	detail, err := s.repository.GetByID(ctx, interviewID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, ErrInterviewNotFound) {
			return nil, ErrInterviewNotFound
		}
		return nil, err
	}

	for _, question := range detail.Questions {
		if question.ID != questionID {
			continue
		}
		language := detail.QuestionLanguage
		if language == "" {
			language = model.QuestionLanguageZhTW
		}
		audio, err := s.generator.GenerateQuestionSpeech(ctx, model.QuestionTTSInput{
			InterviewID:      interviewID,
			QuestionID:       questionID,
			QuestionText:     question.Text,
			QuestionLanguage: language,
		})
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrQuestionTTSUnavailable, err)
		}
		return audio, nil
	}

	return nil, ErrQuestionNotFoundForInterview
}
