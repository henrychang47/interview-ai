package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"interview-ai/backend/internal/model"
	"interview-ai/backend/internal/service"

	"github.com/go-chi/chi/v5"
)

type InterviewService interface {
	CreateInterview(ctx context.Context, input model.CreateInterviewRequest) (model.CreateInterviewResponse, error)
}

func NewInterviewHandler(interviewService InterviewService) http.Handler {
	router := chi.NewRouter()
	router.Post("/", createInterview(interviewService))
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
				log.Printf("create interview: %v", err)
				writeError(w, http.StatusInternalServerError, "failed to create interview")
			}
			return
		}

		writeJSON(w, http.StatusCreated, response)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
