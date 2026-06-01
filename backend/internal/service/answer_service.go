package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	"interview-ai/backend/internal/model"
)

var (
	ErrAudioFileRequired     = errors.New("audio file is required")
	ErrUnsupportedAudioType  = errors.New("audio file must be audio/webm")
	ErrSaveAnswerAudioFailed = errors.New("failed to save answer audio")
)

type AudioStorage interface {
	SaveAnswerAudio(ctx context.Context, interviewID string, questionID string, file io.Reader) (string, error)
}

type AnswerRepository interface {
	EnsureQuestionForInterview(ctx context.Context, interviewID string, questionID string) error
	UpsertAnswer(ctx context.Context, interviewID string, questionID string, audioPath string) (model.Answer, error)
	CompleteInterviewIfAllQuestionsAnswered(ctx context.Context, interviewID string) error
}

type UploadAnswerInput struct {
	InterviewID string
	QuestionID  string
	ContentType string
	Audio       io.Reader
}

type AnswerService struct {
	storage       AudioStorage
	repository    AnswerRepository
	analysisQueue AnswerAnalysisQueue
}

func NewAnswerService(storage AudioStorage, repository AnswerRepository, analysisQueue ...AnswerAnalysisQueue) *AnswerService {
	var queue AnswerAnalysisQueue
	if len(analysisQueue) > 0 {
		queue = analysisQueue[0]
	}
	return &AnswerService{storage: storage, repository: repository, analysisQueue: queue}
}

func (s *AnswerService) UploadAnswer(ctx context.Context, input UploadAnswerInput) (model.UploadAnswerResponse, error) {
	if input.Audio == nil {
		return model.UploadAnswerResponse{}, ErrAudioFileRequired
	}
	if input.ContentType != "audio/webm" {
		return model.UploadAnswerResponse{}, ErrUnsupportedAudioType
	}

	if err := s.repository.EnsureQuestionForInterview(ctx, input.InterviewID, input.QuestionID); err != nil {
		return model.UploadAnswerResponse{}, err
	}

	audioPath, err := s.storage.SaveAnswerAudio(ctx, input.InterviewID, input.QuestionID, input.Audio)
	if err != nil {
		return model.UploadAnswerResponse{}, fmt.Errorf("%w: %v", ErrSaveAnswerAudioFailed, err)
	}

	answer, err := s.repository.UpsertAnswer(ctx, input.InterviewID, input.QuestionID, audioPath)
	if err != nil {
		return model.UploadAnswerResponse{}, err
	}

	if err := s.repository.CompleteInterviewIfAllQuestionsAnswered(ctx, input.InterviewID); err != nil {
		return model.UploadAnswerResponse{}, err
	}

	if answer.AnalysisStatus == "" {
		answer.AnalysisStatus = model.AnswerAnalysisStatusPending
	}

	if s.analysisQueue != nil && answer.AudioPath != nil {
		s.analysisQueue.Enqueue(AnswerAnalysisJob{
			AnswerID:      answer.ID,
			InterviewID:   answer.InterviewID,
			QuestionID:    answer.QuestionID,
			AudioPath:     *answer.AudioPath,
			AudioMIMEType: input.ContentType,
		})
	}

	response := model.UploadAnswerResponse{
		ID:                     answer.ID,
		InterviewID:            answer.InterviewID,
		QuestionID:             answer.QuestionID,
		TranscriptText:         answer.TranscriptText,
		AnalysisStatus:         answer.AnalysisStatus,
		ImprovementSuggestions: answer.ImprovementSuggestions,
		AnalysisError:          answer.AnalysisError,
	}
	if answer.AudioPath != nil {
		response.AudioPath = *answer.AudioPath
	}
	if answer.AnalyzedAt != nil {
		analyzedAt := answer.AnalyzedAt.Format("2006-01-02T15:04:05Z07:00")
		response.AnalyzedAt = &analyzedAt
	}

	return response, nil
}
