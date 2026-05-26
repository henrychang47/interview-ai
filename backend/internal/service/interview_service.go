package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"

	"github.com/jackc/pgx/v5"
)

var (
	ErrJobTitleRequired       = errors.New("job_title is required")
	ErrJobDescriptionRequired = errors.New("job_description is required")
	ErrUserProfileRequired    = errors.New("user_profile is required")
	ErrQuestionCountRange     = errors.New("question_count must be between 1 and 10")
	ErrInterviewNotFound      = errors.New("interview not found")
)

type InterviewRepository interface {
	CreateWithQuestions(ctx context.Context, input model.CreateInterviewRequest, questions []llm.GeneratedQuestion) (model.CreateInterviewResponse, error)
	GetByID(ctx context.Context, interviewID string) (model.InterviewDetail, error)
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

func (s *InterviewService) GetInterview(ctx context.Context, interviewID string) (model.InterviewDetailResponse, error) {
	detail, err := s.repository.GetByID(ctx, interviewID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, ErrInterviewNotFound) {
			return model.InterviewDetailResponse{}, ErrInterviewNotFound
		}
		return model.InterviewDetailResponse{}, err
	}

	return mapInterviewDetailResponse(detail), nil
}

func mapInterviewDetailResponse(detail model.InterviewDetail) model.InterviewDetailResponse {
	questions := make([]model.QuestionResponse, 0, len(detail.Questions))
	for _, question := range detail.Questions {
		questions = append(questions, model.QuestionResponse{
			ID:    question.ID,
			Order: question.Order,
			Text:  question.Text,
		})
	}

	answers := make([]model.AnswerResponse, 0, len(detail.Answers))
	for _, answer := range detail.Answers {
		answers = append(answers, model.AnswerResponse{
			ID:             answer.ID,
			QuestionID:     answer.QuestionID,
			AudioPath:      answer.AudioPath,
			TranscriptText: answer.TranscriptText,
			CreatedAt:      answer.CreatedAt.Format(time.RFC3339),
		})
	}

	return model.InterviewDetailResponse{
		ID:             detail.ID,
		JobTitle:       detail.JobTitle,
		JobDescription: detail.JobDescription,
		UserProfile:    detail.UserProfile,
		QuestionCount:  detail.QuestionCount,
		Status:         detail.Status,
		Questions:      questions,
		Answers:        answers,
	}
}
