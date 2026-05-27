package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"interview-ai/backend/internal/model"
	"interview-ai/backend/internal/service"

	"github.com/go-chi/chi/v5"
)

type InterviewService interface {
	CreateInterview(ctx context.Context, input model.CreateInterviewRequest) (model.CreateInterviewResponse, error)
	GetInterview(ctx context.Context, interviewID string) (model.InterviewDetailResponse, error)
}

func NewInterviewHandler(interviewService InterviewService, answerService AnswerService) http.Handler {
	router := chi.NewRouter()
	router.Post("/", createInterview(interviewService))
	router.Get("/{interviewID}", getInterview(interviewService))
	router.Post("/{interviewID}/questions/{questionID}/answer", uploadAnswer(answerService))
	return router
}

func createInterview(interviewService InterviewService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input model.CreateInterviewRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON request body")
			return
		}

		response, err := interviewService.CreateInterview(r.Context(), input)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrJobTitleRequired),
				errors.Is(err, service.ErrJobDescriptionRequired),
				errors.Is(err, service.ErrUserProfileRequired),
				errors.Is(err, service.ErrQuestionCountRange):
				writeError(w, http.StatusBadRequest, err.Error())
			default:
				slog.Error("create interview", "error", err)
				writeError(w, http.StatusInternalServerError, "failed to create interview")
			}
			return
		}

		writeJSON(w, http.StatusCreated, response)
	}
}

func getInterview(interviewService InterviewService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		interviewID := chi.URLParam(r, "interviewID")
		response, err := interviewService.GetInterview(r.Context(), interviewID)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInterviewNotFound):
				writeError(w, http.StatusNotFound, err.Error())
			default:
				slog.Error("get interview", "error", err)
				writeError(w, http.StatusInternalServerError, "failed to get interview")
			}
			return
		}

		writeJSON(w, http.StatusOK, response)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("write json response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
