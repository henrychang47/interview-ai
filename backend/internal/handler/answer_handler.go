package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"interview-ai/backend/internal/model"
	"interview-ai/backend/internal/service"

	"github.com/go-chi/chi/v5"
)

const maxAnswerAudioBytes = 20 << 20

type AnswerService interface {
	UploadAnswer(ctx context.Context, input service.UploadAnswerInput) (model.UploadAnswerResponse, error)
}

func uploadAnswer(answerService AnswerService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if answerService == nil {
			writeError(w, http.StatusNotFound, "answer upload is not configured")
			return
		}

		interviewID := chi.URLParam(r, "interviewID")
		questionID := chi.URLParam(r, "questionID")

		if err := r.ParseMultipartForm(maxAnswerAudioBytes); err != nil {
			writeError(w, http.StatusBadRequest, service.ErrAudioFileRequired.Error())
			return
		}

		file, header, err := r.FormFile("audio")
		if err != nil {
			writeError(w, http.StatusBadRequest, service.ErrAudioFileRequired.Error())
			return
		}
		defer file.Close()

		response, err := answerService.UploadAnswer(r.Context(), service.UploadAnswerInput{
			InterviewID: interviewID,
			QuestionID:  questionID,
			ContentType: header.Header.Get("Content-Type"),
			Audio:       http.MaxBytesReader(w, file, maxAnswerAudioBytes),
		})
		if err != nil {
			switch {
			case errors.Is(err, service.ErrAudioFileRequired),
				errors.Is(err, service.ErrUnsupportedAudioType):
				writeError(w, http.StatusBadRequest, err.Error())
			case errors.Is(err, service.ErrInterviewNotFound),
				errors.Is(err, service.ErrQuestionNotFoundForInterview):
				writeError(w, http.StatusNotFound, err.Error())
			case errors.Is(err, service.ErrSaveAnswerAudioFailed):
				slog.Error("save answer audio", "error", err)
				writeError(w, http.StatusInternalServerError, service.ErrSaveAnswerAudioFailed.Error())
			default:
				slog.Error("upload answer", "error", err)
				writeError(w, http.StatusInternalServerError, "failed to save answer")
			}
			return
		}

		writeJSON(w, http.StatusCreated, response)
	}
}
