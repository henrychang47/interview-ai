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
	ErrJobTitleRequired              = errors.New("job_title is required")
	ErrJobDescriptionRequired        = errors.New("job_description is required")
	ErrUserProfileRequired           = errors.New("user_profile is required")
	ErrQuestionCountRange            = errors.New("question_count must be between 1 and 10")
	ErrQuestionLanguageUnsupported   = errors.New("question_language must be zh-TW or en-US")
	ErrInterviewNotReady             = model.ErrInterviewNotReady
	ErrInterviewNotFound             = model.ErrInterviewNotFound
	ErrInterviewCreationLimitReached = model.ErrInterviewCreationLimitReached
	ErrQuestionNotFoundForInterview  = errors.New("question not found for interview")
)

type InterviewRepository interface {
	CreatePendingWithCreationLimit(ctx context.Context, input model.CreateInterviewRequest, clientIPHash string, limit int) (model.CreateInterviewResponse, error)
	SaveGeneratedQuestions(ctx context.Context, interviewID string, questions []llm.GeneratedQuestion) error
	MarkQuestionGenerationFailed(ctx context.Context, interviewID string) error
	GetByID(ctx context.Context, interviewID string) (model.InterviewDetail, error)
	Start(ctx context.Context, interviewID string) (model.CreateInterviewResponse, error)
}

type asyncRunner func(func())

type InterviewService struct {
	generator           llm.QuestionGenerator
	repository          InterviewRepository
	runner              asyncRunner
	creationLimitPer24H int
}

func defaultAsyncRunner(task func()) {
	go task()
}

func NewInterviewService(generator llm.QuestionGenerator, repository InterviewRepository) *InterviewService {
	return NewInterviewServiceWithCreationLimit(generator, repository, 5)
}

func NewInterviewServiceWithRunner(generator llm.QuestionGenerator, repository InterviewRepository, runner asyncRunner) *InterviewService {
	return NewInterviewServiceWithCreationLimitAndRunner(generator, repository, 5, runner)
}

func NewInterviewServiceWithCreationLimit(generator llm.QuestionGenerator, repository InterviewRepository, creationLimitPer24H int) *InterviewService {
	return NewInterviewServiceWithCreationLimitAndRunner(generator, repository, creationLimitPer24H, defaultAsyncRunner)
}

func NewInterviewServiceWithCreationLimitAndRunner(generator llm.QuestionGenerator, repository InterviewRepository, creationLimitPer24H int, runner asyncRunner) *InterviewService {
	if creationLimitPer24H <= 0 {
		creationLimitPer24H = 5
	}
	return &InterviewService{
		generator:           generator,
		repository:          repository,
		runner:              runner,
		creationLimitPer24H: creationLimitPer24H,
	}
}

func (s *InterviewService) CreateInterview(ctx context.Context, input model.CreateInterviewRequest, clientIPHash string) (model.CreateInterviewResponse, error) {
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
	if input.QuestionLanguage == "" {
		input.QuestionLanguage = model.QuestionLanguageZhTW
	}
	if input.QuestionLanguage != model.QuestionLanguageZhTW && input.QuestionLanguage != model.QuestionLanguageEnUS {
		return model.CreateInterviewResponse{}, ErrQuestionLanguageUnsupported
	}

	created, err := s.repository.CreatePendingWithCreationLimit(ctx, input, clientIPHash, s.creationLimitPer24H)
	if err != nil {
		return model.CreateInterviewResponse{}, err
	}

	s.runner(func() {
		generationCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		questions, err := s.generator.GenerateQuestions(generationCtx, llm.GenerateQuestionsInput{
			InterviewID:      created.ID,
			JobTitle:         input.JobTitle,
			JobDescription:   input.JobDescription,
			UserProfile:      input.UserProfile,
			QuestionCount:    input.QuestionCount,
			QuestionLanguage: input.QuestionLanguage,
		})
		if err != nil {
			_ = s.repository.MarkQuestionGenerationFailed(context.Background(), created.ID)
			return
		}
		if err := s.repository.SaveGeneratedQuestions(context.Background(), created.ID, questions); err != nil {
			_ = s.repository.MarkQuestionGenerationFailed(context.Background(), created.ID)
		}
	})

	return created, nil
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

func (s *InterviewService) StartInterview(ctx context.Context, interviewID string) (model.CreateInterviewResponse, error) {
	response, err := s.repository.Start(ctx, interviewID)
	if err != nil {
		if errors.Is(err, ErrInterviewNotReady) {
			return model.CreateInterviewResponse{}, ErrInterviewNotReady
		}
		return model.CreateInterviewResponse{}, err
	}
	return response, nil
}

func mapInterviewDetailResponse(detail model.InterviewDetail) model.InterviewDetailResponse {
	questions := make([]model.QuestionResponse, 0, len(detail.Questions))
	if detail.Status == model.InterviewStatusInProgress || detail.Status == model.InterviewStatusCompleted {
		for _, question := range detail.Questions {
			questions = append(questions, model.QuestionResponse{
				ID:    question.ID,
				Order: question.Order,
				Text:  question.Text,
			})
		}
	}

	answers := make([]model.AnswerResponse, 0, len(detail.Answers))
	for _, answer := range detail.Answers {
		response := model.AnswerResponse{
			ID:                     answer.ID,
			QuestionID:             answer.QuestionID,
			AudioPath:              answer.AudioPath,
			TranscriptText:         answer.TranscriptText,
			AnalysisStatus:         answer.AnalysisStatus,
			ImprovementSuggestions: answer.ImprovementSuggestions,
			AnalysisError:          answer.AnalysisError,
			CreatedAt:              answer.CreatedAt.Format(time.RFC3339),
		}
		if response.AnalysisStatus == "" {
			response.AnalysisStatus = model.AnswerAnalysisStatusPending
		}
		if answer.AnalyzedAt != nil {
			analyzedAt := answer.AnalyzedAt.Format(time.RFC3339)
			response.AnalyzedAt = &analyzedAt
		}
		answers = append(answers, response)
	}

	return model.InterviewDetailResponse{
		ID:               detail.ID,
		JobTitle:         detail.JobTitle,
		JobDescription:   detail.JobDescription,
		UserProfile:      detail.UserProfile,
		QuestionCount:    detail.QuestionCount,
		QuestionLanguage: detail.QuestionLanguage,
		Status:           detail.Status,
		Questions:        questions,
		Answers:          answers,
	}
}
