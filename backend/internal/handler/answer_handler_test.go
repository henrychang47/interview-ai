package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"interview-ai/backend/internal/model"
	"interview-ai/backend/internal/service"
)

func TestUploadAnswerReturnsCreatedResponse(t *testing.T) {
	answerService := &stubAnswerService{
		response: model.UploadAnswerResponse{
			ID:          "answer-id",
			InterviewID: "interview-id",
			QuestionID:  "question-id",
			AudioPath:   "storage/audio/interview-id/question-id.webm",
		},
	}
	handler := NewInterviewHandler(&stubInterviewService{}, answerService)
	body, contentType := multipartBody(t, "audio", "answer.webm", "audio/webm", "webm-bytes")
	request := httptest.NewRequest(http.MethodPost, "/interview-id/questions/question-id/answer", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.Code)
	}
	if answerService.input.InterviewID != "interview-id" {
		t.Fatalf("expected interview id, got %q", answerService.input.InterviewID)
	}
	if answerService.input.QuestionID != "question-id" {
		t.Fatalf("expected question id, got %q", answerService.input.QuestionID)
	}
	if answerService.input.ContentType != "audio/webm" {
		t.Fatalf("expected audio/webm, got %q", answerService.input.ContentType)
	}
	uploadedBytes, err := ioReadAllString(answerService.input.Audio)
	if err != nil {
		t.Fatalf("read uploaded bytes: %v", err)
	}
	if uploadedBytes != "webm-bytes" {
		t.Fatalf("expected uploaded bytes, got %q", uploadedBytes)
	}

	var responseBody model.UploadAnswerResponse
	if err := json.Unmarshal(response.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("expected JSON response, got error: %v", err)
	}
	if responseBody.ID != "answer-id" {
		t.Fatalf("expected answer id, got %q", responseBody.ID)
	}
	if responseBody.AudioPath != "storage/audio/interview-id/question-id.webm" {
		t.Fatalf("expected audio path, got %q", responseBody.AudioPath)
	}
}

func TestUploadAnswerRequiresMultipartAudio(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{}, &stubAnswerService{})
	request := httptest.NewRequest(http.MethodPost, "/interview-id/questions/question-id/answer", strings.NewReader(""))
	request.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusBadRequest, "audio file is required")
}

func TestUploadAnswerRejectsOversizedRequest(t *testing.T) {
	answerService := &stubAnswerService{}
	handler := NewInterviewHandler(&stubInterviewService{}, answerService)
	body, contentType := multipartBodyBytes(t, "audio", "answer.webm", "audio/webm", bytes.Repeat([]byte("a"), maxAnswerAudioBytes+1))
	request := httptest.NewRequest(http.MethodPost, "/interview-id/questions/question-id/answer", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusRequestEntityTooLarge, "audio file is too large")
	if answerService.called {
		t.Fatal("expected answer service not to be called for oversized request")
	}
}

func TestUploadAnswerMapsValidationErrors(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{}, &stubAnswerService{err: service.ErrUnsupportedAudioType})
	body, contentType := multipartBody(t, "audio", "answer.wav", "audio/wav", "wav-bytes")
	request := httptest.NewRequest(http.MethodPost, "/interview-id/questions/question-id/answer", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusBadRequest, "audio file must be audio/webm")
}

func TestUploadAnswerMapsNotFoundErrors(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{}, &stubAnswerService{err: service.ErrQuestionNotFoundForInterview})
	body, contentType := multipartBody(t, "audio", "answer.webm", "audio/webm", "webm-bytes")
	request := httptest.NewRequest(http.MethodPost, "/interview-id/questions/question-id/answer", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusNotFound, "question not found for interview")
}

func TestUploadAnswerMapsStorageErrors(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{}, &stubAnswerService{err: service.ErrSaveAnswerAudioFailed})
	body, contentType := multipartBody(t, "audio", "answer.webm", "audio/webm", "webm-bytes")
	request := httptest.NewRequest(http.MethodPost, "/interview-id/questions/question-id/answer", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusInternalServerError, "failed to save answer audio")
}

func multipartBody(t *testing.T, fieldName string, fileName string, contentType string, content string) (*bytes.Buffer, string) {
	return multipartBodyBytes(t, fieldName, fileName, contentType, []byte(content))
}

func multipartBodyBytes(t *testing.T, fieldName string, fileName string, contentType string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="` + fieldName + `"; filename="` + fileName + `"`},
		"Content-Type":        {contentType},
	})
	if err != nil {
		t.Fatalf("create multipart part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write multipart part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func ioReadAllString(reader io.Reader) (string, error) {
	bytes, err := io.ReadAll(reader)
	return string(bytes), err
}

type stubAnswerService struct {
	input    service.UploadAnswerInput
	response model.UploadAnswerResponse
	err      error
	called   bool
}

func (s *stubAnswerService) UploadAnswer(ctx context.Context, input service.UploadAnswerInput) (model.UploadAnswerResponse, error) {
	s.called = true
	s.input = input
	if s.err != nil {
		return model.UploadAnswerResponse{}, s.err
	}
	return s.response, nil
}
