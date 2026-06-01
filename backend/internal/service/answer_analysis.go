package service

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"interview-ai/backend/internal/model"
)

type AnswerAnalysisInput = model.AnswerAnalysisInput
type AnswerAnalysisResult = model.AnswerAnalysisResult

type AnswerAnalyzer interface {
	AnalyzeAnswer(ctx context.Context, input AnswerAnalysisInput) (AnswerAnalysisResult, error)
}

type AnswerAnalysisRepository interface {
	GetAnswerAnalysisContext(ctx context.Context, interviewID string, questionID string) (model.AnswerAnalysisContext, error)
	MarkAnswerAnalysisProcessing(ctx context.Context, answerID string) error
	MarkAnswerAnalysisCompleted(ctx context.Context, answerID string, transcriptText string, improvementSuggestions string) error
	MarkAnswerAnalysisFailed(ctx context.Context, answerID string, message string) error
}

type AnswerAnalysisJob struct {
	AnswerID      string
	InterviewID   string
	QuestionID    string
	AudioPath     string
	AudioMIMEType string
}

type AnswerAnalysisQueue interface {
	Enqueue(job AnswerAnalysisJob)
}

type AnswerAnalysisWorker struct {
	repository AnswerAnalysisRepository
	analyzer   AnswerAnalyzer
}

func NewAnswerAnalysisWorker(repository AnswerAnalysisRepository, analyzer AnswerAnalyzer) *AnswerAnalysisWorker {
	return &AnswerAnalysisWorker{repository: repository, analyzer: analyzer}
}

func (w *AnswerAnalysisWorker) Process(ctx context.Context, job AnswerAnalysisJob) {
	if w == nil || w.repository == nil || w.analyzer == nil {
		return
	}

	if err := w.repository.MarkAnswerAnalysisProcessing(ctx, job.AnswerID); err != nil {
		slog.Error("mark answer analysis processing", "answer_id", job.AnswerID, "error", err)
		return
	}

	analysisContext, err := w.repository.GetAnswerAnalysisContext(ctx, job.InterviewID, job.QuestionID)
	if err != nil {
		w.markFailed(job.AnswerID, err)
		return
	}

	result, err := w.analyzer.AnalyzeAnswer(ctx, AnswerAnalysisInput{
		AnswerID:       job.AnswerID,
		InterviewID:    job.InterviewID,
		QuestionID:     job.QuestionID,
		AudioPath:      job.AudioPath,
		AudioMIMEType:  job.AudioMIMEType,
		JobTitle:       analysisContext.JobTitle,
		JobDescription: analysisContext.JobDescription,
		UserProfile:    analysisContext.UserProfile,
		QuestionText:   analysisContext.QuestionText,
	})
	if err != nil {
		w.markFailed(job.AnswerID, err)
		return
	}

	if err := w.repository.MarkAnswerAnalysisCompleted(
		context.Background(),
		job.AnswerID,
		result.TranscriptText,
		result.ImprovementSuggestions,
	); err != nil {
		slog.Error("mark answer analysis completed", "answer_id", job.AnswerID, "error", err)
	}
}

func (w *AnswerAnalysisWorker) markFailed(answerID string, err error) {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = "answer analysis failed"
	}
	if markErr := w.repository.MarkAnswerAnalysisFailed(context.Background(), answerID, message); markErr != nil {
		slog.Error("mark answer analysis failed", "answer_id", answerID, "error", markErr)
	}
}

type BackgroundAnswerAnalysisQueue struct {
	jobs   chan AnswerAnalysisJob
	worker *AnswerAnalysisWorker
}

func NewBackgroundAnswerAnalysisQueue(ctx context.Context, repository AnswerAnalysisRepository, analyzer AnswerAnalyzer) *BackgroundAnswerAnalysisQueue {
	queue := &BackgroundAnswerAnalysisQueue{
		jobs:   make(chan AnswerAnalysisJob, 32),
		worker: NewAnswerAnalysisWorker(repository, analyzer),
	}

	go queue.run(ctx)
	return queue
}

func (q *BackgroundAnswerAnalysisQueue) Enqueue(job AnswerAnalysisJob) {
	if q == nil {
		return
	}

	select {
	case q.jobs <- job:
	default:
		go q.worker.Process(context.Background(), job)
	}
}

func (q *BackgroundAnswerAnalysisQueue) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-q.jobs:
			processCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			q.worker.Process(processCtx, job)
			cancel()
		}
	}
}
