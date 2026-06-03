package handler

import (
	"context"
	"encoding/base64"
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
	StartInterview(ctx context.Context, interviewID string) (model.CreateInterviewResponse, error)
}

type QuestionTTSService interface {
	GenerateQuestionSpeech(ctx context.Context, interviewID string, questionID string) ([]byte, error)
	GenerateInterviewQuestionSpeech(ctx context.Context, interviewID string) ([]model.QuestionTTSAudio, error)
}

func NewInterviewHandler(interviewService InterviewService, answerService AnswerService) http.Handler {
	return NewInterviewHandlerWithTTS(interviewService, answerService, nil)
}

func NewInterviewHandlerWithTTS(interviewService InterviewService, answerService AnswerService, questionTTSService QuestionTTSService) http.Handler {
	router := chi.NewRouter()
	router.Post("/", createInterview(interviewService))
	router.Get("/{interviewID}", getInterview(interviewService))
	router.Post("/{interviewID}/start", startInterview(interviewService))
	if questionTTSService != nil {
		router.Post("/{interviewID}/questions/tts", generateInterviewQuestionTTS(questionTTSService))
		router.Post("/{interviewID}/questions/{questionID}/tts", generateQuestionTTS(questionTTSService))
	}
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
				errors.Is(err, service.ErrQuestionCountRange),
				errors.Is(err, service.ErrQuestionLanguageUnsupported):
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

func startInterview(interviewService InterviewService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		interviewID := chi.URLParam(r, "interviewID")
		response, err := interviewService.StartInterview(r.Context(), interviewID)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInterviewNotFound):
				writeError(w, http.StatusNotFound, err.Error())
			case errors.Is(err, service.ErrInterviewNotReady):
				writeError(w, http.StatusConflict, err.Error())
			default:
				slog.Error("start interview", "error", err)
				writeError(w, http.StatusInternalServerError, "failed to start interview")
			}
			return
		}

		writeJSON(w, http.StatusOK, response)
	}
}

func generateQuestionTTS(questionTTSService QuestionTTSService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		interviewID := chi.URLParam(r, "interviewID")
		questionID := chi.URLParam(r, "questionID")
		audio, err := questionTTSService.GenerateQuestionSpeech(r.Context(), interviewID, questionID)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInterviewNotFound),
				errors.Is(err, service.ErrQuestionNotFoundForInterview):
				writeError(w, http.StatusNotFound, err.Error())
			case errors.Is(err, service.ErrQuestionTTSUnavailable):
				writeError(w, http.StatusServiceUnavailable, service.ErrQuestionTTSUnavailable.Error())
			default:
				slog.Error("generate question tts", "error", err)
				writeError(w, http.StatusInternalServerError, "failed to generate question audio")
			}
			return
		}

		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(audio); err != nil {
			slog.Error("write question tts audio", "error", err)
		}
	}
}

func generateInterviewQuestionTTS(questionTTSService QuestionTTSService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		interviewID := chi.URLParam(r, "interviewID")
		audio, err := questionTTSService.GenerateInterviewQuestionSpeech(r.Context(), interviewID)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInterviewNotFound):
				writeError(w, http.StatusNotFound, err.Error())
			case errors.Is(err, service.ErrQuestionTTSUnavailable):
				writeError(w, http.StatusServiceUnavailable, service.ErrQuestionTTSUnavailable.Error())
			default:
				slog.Error("generate interview question tts", "error", err)
				writeError(w, http.StatusInternalServerError, "failed to generate question audio")
			}
			return
		}

		responseAudio := make([]model.QuestionTTSAudioResponse, 0, len(audio))
		for _, item := range audio {
			responseAudio = append(responseAudio, model.QuestionTTSAudioResponse{
				QuestionID:  item.QuestionID,
				ContentType: "audio/wav",
				AudioBase64: base64.StdEncoding.EncodeToString(item.Audio),
			})
		}

		writeJSON(w, http.StatusOK, model.InterviewQuestionTTSResponse{Audio: responseAudio})
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
