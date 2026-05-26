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
			Status: model.InterviewStatusQuestionsReady,
		},
	})
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
	if body.Status != model.InterviewStatusQuestionsReady {
		t.Fatalf("expected questions_ready, got %q", body.Status)
	}
}

func TestCreateInterviewRejectsInvalidJSON(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{})
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusBadRequest, "invalid JSON request body")
}

func TestCreateInterviewReturnsValidationError(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{err: service.ErrJobTitleRequired})
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
	handler := NewInterviewHandler(&stubInterviewService{err: errors.New("db failed")})
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
	response model.CreateInterviewResponse
	err      error
}

func (s *stubInterviewService) CreateInterview(ctx context.Context, input model.CreateInterviewRequest) (model.CreateInterviewResponse, error) {
	return s.response, s.err
}
