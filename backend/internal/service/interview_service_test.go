package service

import (
	"context"
	"errors"
	"testing"

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
	input     model.CreateInterviewRequest
	questions []llm.GeneratedQuestion
}

func (s *stubInterviewRepository) CreateWithQuestions(ctx context.Context, input model.CreateInterviewRequest, questions []llm.GeneratedQuestion) (model.CreateInterviewResponse, error) {
	s.input = input
	s.questions = questions
	return model.CreateInterviewResponse{
		ID:     "interview-id",
		Status: model.InterviewStatusQuestionsReady,
	}, nil
}
