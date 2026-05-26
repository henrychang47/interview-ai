package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"
)

func TestCreateInterviewGeneratesQuestionsAndPersistsInterview(t *testing.T) {
	generator := &stubQuestionGenerator{}
	repository := &stubInterviewRepository{}
	service := NewInterviewService(generator, repository)

	response, err := service.CreateInterview(context.Background(), model.CreateInterviewRequest{
		JobTitle:       " 後端工程師 ",
		JobDescription: " 需要熟悉 Go、PostgreSQL、REST API ",
		UserProfile:    " 有 Java 和 Go 學習經驗 ",
		QuestionCount:  3,
	})

	if err != nil {
		t.Fatalf("CreateInterview returned error: %v", err)
	}
	if response.ID != "interview-id" {
		t.Fatalf("expected interview id, got %q", response.ID)
	}
	if response.Status != model.InterviewStatusQuestionsReady {
		t.Fatalf("expected questions_ready, got %q", response.Status)
	}
	if generator.input.JobTitle != "後端工程師" {
		t.Fatalf("expected trimmed job title, got %q", generator.input.JobTitle)
	}
	if len(repository.questions) != 3 {
		t.Fatalf("expected 3 persisted questions, got %d", len(repository.questions))
	}
}

func TestCreateInterviewRequiresJobTitle(t *testing.T) {
	service := NewInterviewService(&stubQuestionGenerator{}, &stubInterviewRepository{})

	_, err := service.CreateInterview(context.Background(), validCreateInterviewRequest(func(input *model.CreateInterviewRequest) {
		input.JobTitle = " "
	}))

	if !errors.Is(err, ErrJobTitleRequired) {
		t.Fatalf("expected ErrJobTitleRequired, got %v", err)
	}
}

func TestCreateInterviewRequiresQuestionCountBetweenOneAndTen(t *testing.T) {
	service := NewInterviewService(&stubQuestionGenerator{}, &stubInterviewRepository{})

	for _, count := range []int{0, 11} {
		_, err := service.CreateInterview(context.Background(), validCreateInterviewRequest(func(input *model.CreateInterviewRequest) {
			input.QuestionCount = count
		}))
		if !errors.Is(err, ErrQuestionCountRange) {
			t.Fatalf("expected ErrQuestionCountRange for %d, got %v", count, err)
		}
	}
}

func TestGetInterviewReturnsDetailResponse(t *testing.T) {
	audioPath := "storage/audio/interview-id/question-id.webm"
	createdAt := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	repository := &stubInterviewRepository{
		detail: model.InterviewDetail{
			ID:             "interview-id",
			JobTitle:       "後端工程師",
			JobDescription: "需要熟悉 Go",
			UserProfile:    "有 Go 學習經驗",
			QuestionCount:  1,
			Status:         model.InterviewStatusQuestionsReady,
			Questions: []model.Question{
				{ID: "question-id", InterviewID: "interview-id", Order: 1, Text: "問題一"},
			},
			Answers: []model.Answer{
				{ID: "answer-id", InterviewID: "interview-id", QuestionID: "question-id", AudioPath: &audioPath, TranscriptText: nil, CreatedAt: createdAt},
			},
		},
	}
	service := NewInterviewService(&stubQuestionGenerator{}, repository)

	response, err := service.GetInterview(context.Background(), "interview-id")

	if err != nil {
		t.Fatalf("GetInterview returned error: %v", err)
	}
	if repository.requestedID != "interview-id" {
		t.Fatalf("expected repository lookup id, got %q", repository.requestedID)
	}
	if response.ID != "interview-id" {
		t.Fatalf("expected interview id, got %q", response.ID)
	}
	if len(response.Questions) != 1 || response.Questions[0].Text != "問題一" {
		t.Fatalf("expected mapped questions, got %+v", response.Questions)
	}
	if len(response.Answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(response.Answers))
	}
	if response.Answers[0].AudioPath == nil || *response.Answers[0].AudioPath != audioPath {
		t.Fatalf("expected mapped audio path, got %+v", response.Answers[0].AudioPath)
	}
	if response.Answers[0].TranscriptText != nil {
		t.Fatalf("expected nil transcript text, got %+v", response.Answers[0].TranscriptText)
	}
	if response.Answers[0].CreatedAt != "2026-05-26T12:00:00Z" {
		t.Fatalf("expected RFC3339 created_at, got %q", response.Answers[0].CreatedAt)
	}
}

func TestGetInterviewReturnsNotFound(t *testing.T) {
	repository := &stubInterviewRepository{getErr: ErrInterviewNotFound}
	service := NewInterviewService(&stubQuestionGenerator{}, repository)

	_, err := service.GetInterview(context.Background(), "missing-id")

	if !errors.Is(err, ErrInterviewNotFound) {
		t.Fatalf("expected ErrInterviewNotFound, got %v", err)
	}
}

func validCreateInterviewRequest(modify func(*model.CreateInterviewRequest)) model.CreateInterviewRequest {
	input := model.CreateInterviewRequest{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go、PostgreSQL、REST API",
		UserProfile:    "有 Java 和 Go 學習經驗",
		QuestionCount:  3,
	}
	modify(&input)
	return input
}

type stubQuestionGenerator struct {
	input llm.GenerateQuestionsInput
}

func (s *stubQuestionGenerator) GenerateQuestions(ctx context.Context, input llm.GenerateQuestionsInput) ([]llm.GeneratedQuestion, error) {
	s.input = input
	questions := make([]llm.GeneratedQuestion, 0, input.QuestionCount)
	for index := 0; index < input.QuestionCount; index++ {
		questions = append(questions, llm.GeneratedQuestion{
			Order: index + 1,
			Text:  "mock question",
		})
	}
	return questions, nil
}

type stubInterviewRepository struct {
	input       model.CreateInterviewRequest
	questions   []llm.GeneratedQuestion
	detail      model.InterviewDetail
	getErr      error
	requestedID string
}

func (s *stubInterviewRepository) CreateWithQuestions(ctx context.Context, input model.CreateInterviewRequest, questions []llm.GeneratedQuestion) (model.CreateInterviewResponse, error) {
	s.input = input
	s.questions = questions
	return model.CreateInterviewResponse{
		ID:     "interview-id",
		Status: model.InterviewStatusQuestionsReady,
	}, nil
}

func (s *stubInterviewRepository) GetByID(ctx context.Context, interviewID string) (model.InterviewDetail, error) {
	s.requestedID = interviewID
	return s.detail, s.getErr
}
