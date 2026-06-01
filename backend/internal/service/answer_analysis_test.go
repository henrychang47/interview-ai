package service

import (
	"context"
	"errors"
	"testing"

	"interview-ai/backend/internal/model"
)

func TestAnswerAnalysisWorkerStoresCompletedAnalysis(t *testing.T) {
	repository := &stubAnswerAnalysisRepository{
		context: model.AnswerAnalysisContext{
			JobTitle:       "後端工程師",
			JobDescription: "需要熟悉 Go、PostgreSQL、REST API",
			UserProfile:    "有 Java 和 Go 學習經驗",
			QuestionText:   "請介紹你做過的 Go API 專案。",
		},
	}
	analyzer := &stubAnswerAnalyzer{
		result: AnswerAnalysisResult{
			TranscriptText:         "這是逐字稿。",
			ImprovementSuggestions: "回答可以更具體。",
		},
	}
	worker := NewAnswerAnalysisWorker(repository, analyzer)

	worker.Process(context.Background(), AnswerAnalysisJob{
		AnswerID:      "answer-id",
		InterviewID:   "interview-id",
		QuestionID:    "question-id",
		AudioPath:     "storage/audio/interview-id/question-id.webm",
		AudioMIMEType: "audio/webm",
	})

	if repository.processingAnswerID != "answer-id" {
		t.Fatalf("expected processing answer id, got %q", repository.processingAnswerID)
	}
	if analyzer.input.AudioPath != "storage/audio/interview-id/question-id.webm" {
		t.Fatalf("expected analyzer audio path, got %q", analyzer.input.AudioPath)
	}
	if analyzer.input.JobTitle != "後端工程師" {
		t.Fatalf("expected analyzer job title, got %q", analyzer.input.JobTitle)
	}
	if analyzer.input.JobDescription != "需要熟悉 Go、PostgreSQL、REST API" {
		t.Fatalf("expected analyzer job description, got %q", analyzer.input.JobDescription)
	}
	if analyzer.input.UserProfile != "有 Java 和 Go 學習經驗" {
		t.Fatalf("expected analyzer user profile, got %q", analyzer.input.UserProfile)
	}
	if analyzer.input.QuestionText != "請介紹你做過的 Go API 專案。" {
		t.Fatalf("expected analyzer question text, got %q", analyzer.input.QuestionText)
	}
	if repository.completedAnswerID != "answer-id" {
		t.Fatalf("expected completed answer id, got %q", repository.completedAnswerID)
	}
	if repository.transcriptText != "這是逐字稿。" {
		t.Fatalf("expected transcript text, got %q", repository.transcriptText)
	}
	if repository.improvementSuggestions != "回答可以更具體。" {
		t.Fatalf("expected improvement suggestions, got %q", repository.improvementSuggestions)
	}
	if repository.failedAnswerID != "" {
		t.Fatalf("expected no failed answer id, got %q", repository.failedAnswerID)
	}
}

func TestAnswerAnalysisWorkerStoresFailedAnalysis(t *testing.T) {
	repository := &stubAnswerAnalysisRepository{
		context: model.AnswerAnalysisContext{
			JobTitle:       "後端工程師",
			JobDescription: "需要熟悉 Go",
			UserProfile:    "有 Go 學習經驗",
			QuestionText:   "問題一",
		},
	}
	worker := NewAnswerAnalysisWorker(repository, &stubAnswerAnalyzer{err: errors.New("gemini unavailable")})

	worker.Process(context.Background(), AnswerAnalysisJob{
		AnswerID:      "answer-id",
		InterviewID:   "interview-id",
		QuestionID:    "question-id",
		AudioPath:     "storage/audio/interview-id/question-id.webm",
		AudioMIMEType: "audio/webm",
	})

	if repository.processingAnswerID != "answer-id" {
		t.Fatalf("expected processing answer id, got %q", repository.processingAnswerID)
	}
	if repository.failedAnswerID != "answer-id" {
		t.Fatalf("expected failed answer id, got %q", repository.failedAnswerID)
	}
	if repository.analysisError != "gemini unavailable" {
		t.Fatalf("expected analysis error, got %q", repository.analysisError)
	}
	if repository.completedAnswerID != "" {
		t.Fatalf("expected no completed answer id, got %q", repository.completedAnswerID)
	}
}

func TestAnswerAnalysisWorkerMarksFailedWhenContextLookupFails(t *testing.T) {
	repository := &stubAnswerAnalysisRepository{contextErr: errors.New("context not found")}
	analyzer := &stubAnswerAnalyzer{}
	worker := NewAnswerAnalysisWorker(repository, analyzer)

	worker.Process(context.Background(), AnswerAnalysisJob{
		AnswerID:      "answer-id",
		InterviewID:   "interview-id",
		QuestionID:    "question-id",
		AudioPath:     "storage/audio/interview-id/question-id.webm",
		AudioMIMEType: "audio/webm",
	})

	if repository.processingAnswerID != "answer-id" {
		t.Fatalf("expected processing answer id, got %q", repository.processingAnswerID)
	}
	if repository.failedAnswerID != "answer-id" {
		t.Fatalf("expected failed answer id, got %q", repository.failedAnswerID)
	}
	if repository.analysisError != "context not found" {
		t.Fatalf("expected context lookup error, got %q", repository.analysisError)
	}
	if analyzer.called {
		t.Fatal("expected analyzer not to be called when context lookup fails")
	}
}

type stubAnswerAnalyzer struct {
	input  AnswerAnalysisInput
	result AnswerAnalysisResult
	err    error
	called bool
}

func (a *stubAnswerAnalyzer) AnalyzeAnswer(ctx context.Context, input AnswerAnalysisInput) (AnswerAnalysisResult, error) {
	a.called = true
	a.input = input
	if a.err != nil {
		return AnswerAnalysisResult{}, a.err
	}
	return a.result, nil
}

type stubAnswerAnalysisRepository struct {
	processingAnswerID     string
	completedAnswerID      string
	failedAnswerID         string
	transcriptText         string
	improvementSuggestions string
	analysisError          string
	context                model.AnswerAnalysisContext
	contextErr             error
}

func (r *stubAnswerAnalysisRepository) MarkAnswerAnalysisProcessing(ctx context.Context, answerID string) error {
	r.processingAnswerID = answerID
	return nil
}

func (r *stubAnswerAnalysisRepository) GetAnswerAnalysisContext(ctx context.Context, interviewID string, questionID string) (model.AnswerAnalysisContext, error) {
	if r.contextErr != nil {
		return model.AnswerAnalysisContext{}, r.contextErr
	}
	return r.context, nil
}

func (r *stubAnswerAnalysisRepository) MarkAnswerAnalysisCompleted(ctx context.Context, answerID string, transcriptText string, improvementSuggestions string) error {
	r.completedAnswerID = answerID
	r.transcriptText = transcriptText
	r.improvementSuggestions = improvementSuggestions
	return nil
}

func (r *stubAnswerAnalysisRepository) MarkAnswerAnalysisFailed(ctx context.Context, answerID string, message string) error {
	r.failedAnswerID = answerID
	r.analysisError = message
	return nil
}
