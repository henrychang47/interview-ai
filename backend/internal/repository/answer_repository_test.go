package repository

import (
	"context"
	"errors"
	"os"
	"testing"

	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"
	"interview-ai/backend/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUpsertAnswerCreatesAndUpdatesAnswer(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	interviewRepository := NewInterviewRepository(pool)
	created, err := interviewRepository.CreateWithQuestions(ctx, model.CreateInterviewRequest{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go、PostgreSQL、REST API",
		UserProfile:    "有 Go 學習經驗",
		QuestionCount:  1,
	}, []llm.GeneratedQuestion{{Order: 1, Text: "問題一"}})
	if err != nil {
		t.Fatalf("CreateWithQuestions returned error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", created.ID)
	})

	var questionID string
	if err := pool.QueryRow(ctx, `
		SELECT id
		FROM questions
		WHERE interview_id = $1 AND question_order = 1
	`, created.ID).Scan(&questionID); err != nil {
		t.Fatalf("query question id: %v", err)
	}

	repository := NewAnswerRepository(pool)
	answer, err := repository.UpsertAnswer(ctx, created.ID, questionID, "storage/audio/first.webm")
	if err != nil {
		t.Fatalf("UpsertAnswer returned error: %v", err)
	}
	if answer.ID == "" {
		t.Fatal("expected answer id")
	}
	if answer.InterviewID != created.ID {
		t.Fatalf("expected interview id %q, got %q", created.ID, answer.InterviewID)
	}
	if answer.QuestionID != questionID {
		t.Fatalf("expected question id %q, got %q", questionID, answer.QuestionID)
	}
	if answer.AudioPath == nil || *answer.AudioPath != "storage/audio/first.webm" {
		t.Fatalf("expected first audio path, got %+v", answer.AudioPath)
	}
	if answer.TranscriptText != nil {
		t.Fatalf("expected nil transcript text, got %+v", answer.TranscriptText)
	}

	updated, err := repository.UpsertAnswer(ctx, created.ID, questionID, "storage/audio/second.webm")
	if err != nil {
		t.Fatalf("second UpsertAnswer returned error: %v", err)
	}
	if updated.ID != answer.ID {
		t.Fatalf("expected same answer id after upsert, got %q then %q", answer.ID, updated.ID)
	}
	if updated.AudioPath == nil || *updated.AudioPath != "storage/audio/second.webm" {
		t.Fatalf("expected updated audio path, got %+v", updated.AudioPath)
	}

	var answerCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM answers
		WHERE interview_id = $1 AND question_id = $2
	`, created.ID, questionID).Scan(&answerCount); err != nil {
		t.Fatalf("count answers: %v", err)
	}
	if answerCount != 1 {
		t.Fatalf("expected one answer row after upsert, got %d", answerCount)
	}
}

func TestUpsertAnswerRejectsMissingInterview(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	repository := NewAnswerRepository(pool)
	_, err = repository.UpsertAnswer(ctx, "00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000001", "storage/audio/missing.webm")

	if !errors.Is(err, service.ErrInterviewNotFound) {
		t.Fatalf("expected ErrInterviewNotFound, got %v", err)
	}
}

func TestUpsertAnswerRejectsQuestionOutsideInterview(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	interviewRepository := NewInterviewRepository(pool)
	first, err := interviewRepository.CreateWithQuestions(ctx, model.CreateInterviewRequest{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go",
		UserProfile:    "有 Go 學習經驗",
		QuestionCount:  1,
	}, []llm.GeneratedQuestion{{Order: 1, Text: "第一場問題"}})
	if err != nil {
		t.Fatalf("create first interview: %v", err)
	}
	second, err := interviewRepository.CreateWithQuestions(ctx, model.CreateInterviewRequest{
		JobTitle:       "前端工程師",
		JobDescription: "需要熟悉 React",
		UserProfile:    "有 React 學習經驗",
		QuestionCount:  1,
	}, []llm.GeneratedQuestion{{Order: 1, Text: "第二場問題"}})
	if err != nil {
		t.Fatalf("create second interview: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", first.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", second.ID)
	})

	var secondQuestionID string
	if err := pool.QueryRow(ctx, `
		SELECT id
		FROM questions
		WHERE interview_id = $1 AND question_order = 1
	`, second.ID).Scan(&secondQuestionID); err != nil {
		t.Fatalf("query second question id: %v", err)
	}

	repository := NewAnswerRepository(pool)
	_, err = repository.UpsertAnswer(ctx, first.ID, secondQuestionID, "storage/audio/wrong-question.webm")

	if !errors.Is(err, service.ErrQuestionNotFoundForInterview) {
		t.Fatalf("expected ErrQuestionNotFoundForInterview, got %v", err)
	}
}
