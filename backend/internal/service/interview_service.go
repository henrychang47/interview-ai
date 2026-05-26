package service

import (
	"context"
	"errors"
	"strings"

	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"
)

var (
	ErrJobTitleRequired       = errors.New("job_title is required")
	ErrJobDescriptionRequired = errors.New("job_description is required")
	ErrUserProfileRequired    = errors.New("user_profile is required")
	ErrQuestionCountRange     = errors.New("question_count must be between 1 and 10")
)

type InterviewRepository interface {
	CreateWithQuestions(ctx context.Context, input model.CreateInterviewRequest, questions []llm.GeneratedQuestion) (model.CreateInterviewResponse, error)
}

type InterviewService struct {
	generator  llm.QuestionGenerator
	repository InterviewRepository
}

func NewInterviewService(generator llm.QuestionGenerator, repository InterviewRepository) *InterviewService {
	return &InterviewService{generator: generator, repository: repository}
}

func (s *InterviewService) CreateInterview(ctx context.Context, input model.CreateInterviewRequest) (model.CreateInterviewResponse, error) {
	input.JobTitle = strings.TrimSpace(input.JobTitle)
	input.JobDescription = strings.TrimSpace(input.JobDescription)
	input.UserProfile = strings.TrimSpace(input.UserProfile)

	if input.JobTitle == "" {
		return model.CreateInterviewResponse{}, ErrJobTitleRequired
	}
	if input.JobDescription == "" {
		return model.CreateInterviewResponse{}, ErrJobDescriptionRequired
	}
	if input.UserProfile == "" {
		return model.CreateInterviewResponse{}, ErrUserProfileRequired
	}
	if input.QuestionCount < 1 || input.QuestionCount > 10 {
		return model.CreateInterviewResponse{}, ErrQuestionCountRange
	}

	questions, err := s.generator.GenerateQuestions(ctx, llm.GenerateQuestionsInput{
		JobTitle:       input.JobTitle,
		JobDescription: input.JobDescription,
		UserProfile:    input.UserProfile,
		QuestionCount:  input.QuestionCount,
	})
	if err != nil {
		return model.CreateInterviewResponse{}, err
	}

	return s.repository.CreateWithQuestions(ctx, input, questions)
}
