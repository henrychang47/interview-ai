package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"interview-ai/backend/internal/model"
	"interview-ai/backend/internal/service"
)

func TestCreateInterviewReturnsCreatedResponse(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{
		response: model.CreateInterviewResponse{
			ID:     "interview-id",
			Status: model.InterviewStatusGeneratingQuestions,
		},
	}, nil)
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{
		"job_title":"後端工程師",
		"job_description":"需要熟悉 Go、PostgreSQL、REST API",
		"user_profile":"有 Java 和 Go 學習經驗",
		"question_count":3
	}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.Code)
	}
	var body model.CreateInterviewResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON response, got error: %v", err)
	}
	if body.ID != "interview-id" {
		t.Fatalf("expected interview id, got %q", body.ID)
	}
	if body.Status != model.InterviewStatusGeneratingQuestions {
		t.Fatalf("expected generating_questions, got %q", body.Status)
	}
}

func TestCreateInterviewRejectsInvalidJSON(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{}, nil)
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusBadRequest, "invalid JSON request body")
}

func TestCreateInterviewReturnsValidationError(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{err: service.ErrJobTitleRequired}, nil)
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{
		"job_title":"",
		"job_description":"需要熟悉 Go",
		"user_profile":"有 Go 學習經驗",
		"question_count":3
	}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusBadRequest, "job_title is required")
}

func TestCreateInterviewReturnsServerError(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{err: errors.New("db failed")}, nil)
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{
		"job_title":"後端工程師",
		"job_description":"需要熟悉 Go",
		"user_profile":"有 Go 學習經驗",
		"question_count":3
	}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusInternalServerError, "failed to create interview")
}

func TestGetInterviewReturnsDetail(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{
		detailResponse: model.InterviewDetailResponse{
			ID:             "interview-id",
			JobTitle:       "後端工程師",
			JobDescription: "需要熟悉 Go",
			UserProfile:    "有 Go 學習經驗",
			QuestionCount:  1,
			Status:         model.InterviewStatusQuestionsReady,
			Questions: []model.QuestionResponse{
				{ID: "question-id", Order: 1, Text: "問題一"},
			},
			Answers: []model.AnswerResponse{},
		},
	}, nil)
	request := httptest.NewRequest(http.MethodGet, "/interview-id", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var body model.InterviewDetailResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON response, got error: %v", err)
	}
	if body.ID != "interview-id" {
		t.Fatalf("expected interview id, got %q", body.ID)
	}
	if len(body.Questions) != 1 || body.Questions[0].Text != "問題一" {
		t.Fatalf("expected questions response, got %+v", body.Questions)
	}
}

func TestGetInterviewReturnsNotFound(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{getErr: service.ErrInterviewNotFound}, nil)
	request := httptest.NewRequest(http.MethodGet, "/missing-id", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusNotFound, "interview not found")
}

func TestGetInterviewReturnsServerError(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{getErr: errors.New("db failed")}, nil)
	request := httptest.NewRequest(http.MethodGet, "/interview-id", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusInternalServerError, "failed to get interview")
}

func TestStartInterviewReturnsInProgress(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{
		startResponse: model.CreateInterviewResponse{ID: "interview-id", Status: model.InterviewStatusInProgress},
	}, nil)
	request := httptest.NewRequest(http.MethodPost, "/interview-id/start", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var body model.CreateInterviewResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON response, got error: %v", err)
	}
	if body.ID != "interview-id" || body.Status != model.InterviewStatusInProgress {
		t.Fatalf("unexpected start response: %+v", body)
	}
}

func TestStartInterviewReturnsNotReady(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{startErr: service.ErrInterviewNotReady}, nil)
	request := httptest.NewRequest(http.MethodPost, "/interview-id/start", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusConflict, "interview is not ready to start")
}

func TestGenerateQuestionTTSReturnsWAVAudio(t *testing.T) {
	handler := NewInterviewHandlerWithTTS(&stubInterviewService{}, nil, &stubQuestionTTSService{
		audio: []byte("wav-bytes"),
	})
	request := httptest.NewRequest(http.MethodPost, "/interview-id/questions/question-id/tts", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if response.Header().Get("Content-Type") != "audio/wav" {
		t.Fatalf("expected audio/wav content type, got %q", response.Header().Get("Content-Type"))
	}
	if response.Body.String() != "wav-bytes" {
		t.Fatalf("expected audio response body, got %q", response.Body.String())
	}
}

func TestGenerateQuestionTTSReturnsUnavailableForFallback(t *testing.T) {
	handler := NewInterviewHandlerWithTTS(&stubInterviewService{}, nil, &stubQuestionTTSService{
		err: service.ErrQuestionTTSUnavailable,
	})
	request := httptest.NewRequest(http.MethodPost, "/interview-id/questions/question-id/tts", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusServiceUnavailable, "question TTS is unavailable")
}

func TestGenerateQuestionTTSReturnsMissingQuestion(t *testing.T) {
	handler := NewInterviewHandlerWithTTS(&stubInterviewService{}, nil, &stubQuestionTTSService{
		err: service.ErrQuestionNotFoundForInterview,
	})
	request := httptest.NewRequest(http.MethodPost, "/interview-id/questions/question-id/tts", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusNotFound, "question not found for interview")
}

func assertErrorResponse(t *testing.T, response *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("expected status %d, got %d", status, response.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON error response, got error: %v", err)
	}
	if body["error"] != message {
		t.Fatalf("expected error %q, got %q", message, body["error"])
	}
}

type stubInterviewService struct {
	response       model.CreateInterviewResponse
	err            error
	detailResponse model.InterviewDetailResponse
	getErr         error
	startResponse  model.CreateInterviewResponse
	startErr       error
}

func (s *stubInterviewService) CreateInterview(ctx context.Context, input model.CreateInterviewRequest) (model.CreateInterviewResponse, error) {
	return s.response, s.err
}

func (s *stubInterviewService) GetInterview(ctx context.Context, interviewID string) (model.InterviewDetailResponse, error) {
	return s.detailResponse, s.getErr
}

func (s *stubInterviewService) StartInterview(ctx context.Context, interviewID string) (model.CreateInterviewResponse, error) {
	return s.startResponse, s.startErr
}

type stubQuestionTTSService struct {
	audio []byte
	err   error
}

func (s *stubQuestionTTSService) GenerateQuestionSpeech(ctx context.Context, interviewID string, questionID string) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.audio, nil
}
