package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"
)

func TestCreateInterviewCreatesGeneratingInterviewAndStartsQuestionGeneration(t *testing.T) {
	generator := &stubQuestionGenerator{}
	repository := &stubInterviewRepository{}
	service := NewInterviewServiceWithRunner(generator, repository, func(task func()) { task() })

	response, err := service.CreateInterview(context.Background(), model.CreateInterviewRequest{
		JobTitle:         " 後端工程師 ",
		JobDescription:   " 需要熟悉 Go、PostgreSQL、REST API ",
		UserProfile:      " 有 Java 和 Go 學習經驗 ",
		QuestionCount:    3,
		QuestionLanguage: model.QuestionLanguageZhTW,
	}, "client-ip-hash")

	if err != nil {
		t.Fatalf("CreateInterview returned error: %v", err)
	}
	if response.ID != "interview-id" {
		t.Fatalf("expected interview id, got %q", response.ID)
	}
	if response.Status != model.InterviewStatusGeneratingQuestions {
		t.Fatalf("expected generating_questions, got %q", response.Status)
	}
	if generator.input.JobTitle != "後端工程師" {
		t.Fatalf("expected trimmed job title, got %q", generator.input.JobTitle)
	}
	if generator.input.InterviewID != "interview-id" {
		t.Fatalf("expected generator interview id, got %q", generator.input.InterviewID)
	}
	if generator.input.QuestionLanguage != model.QuestionLanguageZhTW {
		t.Fatalf("expected zh-TW generator language, got %q", generator.input.QuestionLanguage)
	}
	if len(repository.savedQuestions) != 3 {
		t.Fatalf("expected 3 generated questions saved, got %d", len(repository.savedQuestions))
	}
	if repository.savedQuestionsInterviewID != "interview-id" {
		t.Fatalf("expected generated questions saved for interview-id, got %q", repository.savedQuestionsInterviewID)
	}
	if repository.clientIPHash != "client-ip-hash" {
		t.Fatalf("expected creation limit client IP hash, got %q", repository.clientIPHash)
	}
	if repository.creationLimit != 5 {
		t.Fatalf("expected default creation limit 5, got %d", repository.creationLimit)
	}
}

func TestCreateInterviewDefaultsQuestionLanguageToZhTW(t *testing.T) {
	generator := &stubQuestionGenerator{}
	repository := &stubInterviewRepository{}
	service := NewInterviewServiceWithRunner(generator, repository, func(task func()) { task() })

	_, err := service.CreateInterview(context.Background(), validCreateInterviewRequest(func(input *model.CreateInterviewRequest) {
		input.QuestionLanguage = ""
	}), "client-ip-hash")

	if err != nil {
		t.Fatalf("CreateInterview returned error: %v", err)
	}
	if repository.input.QuestionLanguage != model.QuestionLanguageZhTW {
		t.Fatalf("expected default language zh-TW, got %q", repository.input.QuestionLanguage)
	}
}

func TestCreateInterviewRejectsUnsupportedQuestionLanguage(t *testing.T) {
	service := NewInterviewServiceWithRunner(&stubQuestionGenerator{}, &stubInterviewRepository{}, func(task func()) { task() })

	_, err := service.CreateInterview(context.Background(), validCreateInterviewRequest(func(input *model.CreateInterviewRequest) {
		input.QuestionLanguage = "ja-JP"
	}), "client-ip-hash")

	if !errors.Is(err, ErrQuestionLanguageUnsupported) {
		t.Fatalf("expected ErrQuestionLanguageUnsupported, got %v", err)
	}
}

func TestCreateInterviewMarksQuestionGenerationFailed(t *testing.T) {
	generator := &stubQuestionGenerator{err: errors.New("generator failed")}
	repository := &stubInterviewRepository{}
	service := NewInterviewServiceWithRunner(generator, repository, func(task func()) { task() })

	_, err := service.CreateInterview(context.Background(), validCreateInterviewRequest(func(input *model.CreateInterviewRequest) {
		input.QuestionLanguage = model.QuestionLanguageEnUS
	}), "client-ip-hash")

	if err != nil {
		t.Fatalf("CreateInterview should return created interview before background failure, got %v", err)
	}
	if repository.failedInterviewID != "interview-id" {
		t.Fatalf("expected interview marked failed, got %q", repository.failedInterviewID)
	}
}

func TestCreateInterviewRequiresJobTitle(t *testing.T) {
	service := NewInterviewService(&stubQuestionGenerator{}, &stubInterviewRepository{})

	_, err := service.CreateInterview(context.Background(), validCreateInterviewRequest(func(input *model.CreateInterviewRequest) {
		input.JobTitle = " "
	}), "client-ip-hash")

	if !errors.Is(err, ErrJobTitleRequired) {
		t.Fatalf("expected ErrJobTitleRequired, got %v", err)
	}
}

func TestCreateInterviewDoesNotConsumeLimitForValidationError(t *testing.T) {
	repository := &stubInterviewRepository{}
	service := NewInterviewService(&stubQuestionGenerator{}, repository)

	_, err := service.CreateInterview(context.Background(), validCreateInterviewRequest(func(input *model.CreateInterviewRequest) {
		input.JobTitle = " "
	}), "client-ip-hash")

	if !errors.Is(err, ErrJobTitleRequired) {
		t.Fatalf("expected ErrJobTitleRequired, got %v", err)
	}
	if repository.createPendingCalled {
		t.Fatal("expected validation error not to create interview or consume creation limit")
	}
}

func TestCreateInterviewReturnsCreationLimitError(t *testing.T) {
	generator := &stubQuestionGenerator{}
	repository := &stubInterviewRepository{createErr: ErrInterviewCreationLimitReached}
	service := NewInterviewServiceWithRunner(generator, repository, func(task func()) { task() })

	_, err := service.CreateInterview(context.Background(), validCreateInterviewRequest(func(input *model.CreateInterviewRequest) {}), "client-ip-hash")

	if !errors.Is(err, ErrInterviewCreationLimitReached) {
		t.Fatalf("expected ErrInterviewCreationLimitReached, got %v", err)
	}
	if generator.input.JobTitle != "" {
		t.Fatal("expected question generation not to start when creation limit is reached")
	}
}

func TestCreateInterviewRequiresQuestionCountBetweenOneAndTen(t *testing.T) {
	service := NewInterviewService(&stubQuestionGenerator{}, &stubInterviewRepository{})

	for _, count := range []int{0, 11} {
		_, err := service.CreateInterview(context.Background(), validCreateInterviewRequest(func(input *model.CreateInterviewRequest) {
			input.QuestionCount = count
		}), "client-ip-hash")
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
			Status:         model.InterviewStatusInProgress,
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

func TestGetInterviewHidesQuestionsBeforeInterviewStarts(t *testing.T) {
	repository := &stubInterviewRepository{
		detail: model.InterviewDetail{
			ID:               "interview-id",
			JobTitle:         "後端工程師",
			JobDescription:   "需要熟悉 Go",
			UserProfile:      "有 Go 經驗",
			QuestionCount:    1,
			QuestionLanguage: model.QuestionLanguageZhTW,
			Status:           model.InterviewStatusQuestionsReady,
			Questions: []model.Question{
				{ID: "question-id", InterviewID: "interview-id", Order: 1, Text: "不能在準備頁顯示"},
			},
		},
	}
	service := NewInterviewServiceWithRunner(&stubQuestionGenerator{}, repository, func(task func()) { task() })

	response, err := service.GetInterview(context.Background(), "interview-id")

	if err != nil {
		t.Fatalf("GetInterview returned error: %v", err)
	}
	if len(response.Questions) != 0 {
		t.Fatalf("expected questions hidden before start, got %+v", response.Questions)
	}
}

func TestGetInterviewShowsQuestionsAfterInterviewStarts(t *testing.T) {
	repository := &stubInterviewRepository{
		detail: model.InterviewDetail{
			ID:               "interview-id",
			JobTitle:         "後端工程師",
			JobDescription:   "需要熟悉 Go",
			UserProfile:      "有 Go 經驗",
			QuestionCount:    1,
			QuestionLanguage: model.QuestionLanguageZhTW,
			Status:           model.InterviewStatusInProgress,
			Questions: []model.Question{
				{ID: "question-id", InterviewID: "interview-id", Order: 1, Text: "開始後顯示"},
			},
		},
	}
	service := NewInterviewServiceWithRunner(&stubQuestionGenerator{}, repository, func(task func()) { task() })

	response, err := service.GetInterview(context.Background(), "interview-id")

	if err != nil {
		t.Fatalf("GetInterview returned error: %v", err)
	}
	if len(response.Questions) != 1 || response.Questions[0].Text != "開始後顯示" {
		t.Fatalf("expected questions after start, got %+v", response.Questions)
	}
}

func TestStartInterviewMarksReadyInterviewInProgress(t *testing.T) {
	repository := &stubInterviewRepository{
		startResponse: model.CreateInterviewResponse{ID: "interview-id", Status: model.InterviewStatusInProgress},
	}
	service := NewInterviewServiceWithRunner(&stubQuestionGenerator{}, repository, func(task func()) { task() })

	response, err := service.StartInterview(context.Background(), "interview-id")

	if err != nil {
		t.Fatalf("StartInterview returned error: %v", err)
	}
	if repository.startedInterviewID != "interview-id" {
		t.Fatalf("expected started interview-id, got %q", repository.startedInterviewID)
	}
	if response.Status != model.InterviewStatusInProgress {
		t.Fatalf("expected in_progress, got %q", response.Status)
	}
}

func TestStartInterviewReturnsNotReady(t *testing.T) {
	repository := &stubInterviewRepository{startErr: ErrInterviewNotReady}
	service := NewInterviewServiceWithRunner(&stubQuestionGenerator{}, repository, func(task func()) { task() })

	_, err := service.StartInterview(context.Background(), "interview-id")

	if !errors.Is(err, ErrInterviewNotReady) {
		t.Fatalf("expected ErrInterviewNotReady, got %v", err)
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
		JobTitle:         "後端工程師",
		JobDescription:   "需要熟悉 Go、PostgreSQL、REST API",
		UserProfile:      "有 Java 和 Go 學習經驗",
		QuestionCount:    3,
		QuestionLanguage: model.QuestionLanguageZhTW,
	}
	modify(&input)
	return input
}

type stubQuestionGenerator struct {
	input llm.GenerateQuestionsInput
	err   error
}

func (s *stubQuestionGenerator) GenerateQuestions(ctx context.Context, input llm.GenerateQuestionsInput) ([]llm.GeneratedQuestion, error) {
	s.input = input
	if s.err != nil {
		return nil, s.err
	}
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
	input                     model.CreateInterviewRequest
	clientIPHash              string
	creationLimit             int
	createErr                 error
	createPendingCalled       bool
	savedQuestions            []llm.GeneratedQuestion
	savedQuestionsInterviewID string
	failedInterviewID         string
	startedInterviewID        string
	startResponse             model.CreateInterviewResponse
	startErr                  error
	detail                    model.InterviewDetail
	getErr                    error
	requestedID               string
}

func (s *stubInterviewRepository) CreatePendingWithCreationLimit(ctx context.Context, input model.CreateInterviewRequest, clientIPHash string, limit int) (model.CreateInterviewResponse, error) {
	s.createPendingCalled = true
	s.input = input
	s.clientIPHash = clientIPHash
	s.creationLimit = limit
	if s.createErr != nil {
		return model.CreateInterviewResponse{}, s.createErr
	}
	return model.CreateInterviewResponse{
		ID:     "interview-id",
		Status: model.InterviewStatusGeneratingQuestions,
	}, nil
}

func (s *stubInterviewRepository) SaveGeneratedQuestions(ctx context.Context, interviewID string, questions []llm.GeneratedQuestion) error {
	s.savedQuestionsInterviewID = interviewID
	s.savedQuestions = questions
	return nil
}

func (s *stubInterviewRepository) MarkQuestionGenerationFailed(ctx context.Context, interviewID string) error {
	s.failedInterviewID = interviewID
	return nil
}

func (s *stubInterviewRepository) Start(ctx context.Context, interviewID string) (model.CreateInterviewResponse, error) {
	s.startedInterviewID = interviewID
	return s.startResponse, s.startErr
}

func (s *stubInterviewRepository) GetByID(ctx context.Context, interviewID string) (model.InterviewDetail, error) {
	s.requestedID = interviewID
	return s.detail, s.getErr
}
