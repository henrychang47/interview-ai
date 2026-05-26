package repository

import (
	"context"
	"os"
	"testing"

	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCreateWithQuestionsPersistsInterviewAndQuestions(t *testing.T) {
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

	repository := NewInterviewRepository(pool)
	response, err := repository.CreateWithQuestions(ctx, model.CreateInterviewRequest{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go、PostgreSQL、REST API",
		UserProfile:    "有 Java 和 Go 學習經驗",
		QuestionCount:  3,
	}, []llm.GeneratedQuestion{
		{Order: 1, Text: "問題一"},
		{Order: 2, Text: "問題二"},
		{Order: 3, Text: "問題三"},
	})
	if err != nil {
		t.Fatalf("CreateWithQuestions returned error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", response.ID)
	})

	if response.ID == "" {
		t.Fatal("expected interview id")
	}
	if response.Status != "questions_ready" {
		t.Fatalf("expected questions_ready, got %q", response.Status)
	}

	var interviewCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM interviews WHERE id = $1", response.ID).Scan(&interviewCount); err != nil {
		t.Fatalf("query interview count: %v", err)
	}
	if interviewCount != 1 {
		t.Fatalf("expected 1 interview row, got %d", interviewCount)
	}

	var questionCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM questions WHERE interview_id = $1", response.ID).Scan(&questionCount); err != nil {
		t.Fatalf("query question count: %v", err)
	}
	if questionCount != 3 {
		t.Fatalf("expected 3 question rows, got %d", questionCount)
	}
}
