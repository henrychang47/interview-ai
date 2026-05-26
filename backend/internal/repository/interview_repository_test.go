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

func TestGetByIDReturnsInterviewQuestionsAndAnswers(t *testing.T) {
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
	created, err := repository.CreateWithQuestions(ctx, model.CreateInterviewRequest{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go、PostgreSQL、REST API",
		UserProfile:    "有 Java 和 Go 學習經驗",
		QuestionCount:  2,
	}, []llm.GeneratedQuestion{
		{Order: 1, Text: "問題一"},
		{Order: 2, Text: "問題二"},
	})
	if err != nil {
		t.Fatalf("CreateWithQuestions returned error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", created.ID)
	})

	var firstQuestionID string
	if err := pool.QueryRow(ctx, `
		SELECT id
		FROM questions
		WHERE interview_id = $1 AND question_order = 1
	`, created.ID).Scan(&firstQuestionID); err != nil {
		t.Fatalf("query first question id: %v", err)
	}

	audioPath := "storage/audio/" + created.ID + "/" + firstQuestionID + ".webm"
	_, err = pool.Exec(ctx, `
		INSERT INTO answers (interview_id, question_id, audio_path, transcript_text)
		VALUES ($1, $2, $3, NULL)
	`, created.ID, firstQuestionID, audioPath)
	if err != nil {
		t.Fatalf("insert answer: %v", err)
	}

	detail, err := repository.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}

	if detail.ID != created.ID {
		t.Fatalf("expected interview id %q, got %q", created.ID, detail.ID)
	}
	if detail.JobTitle != "後端工程師" {
		t.Fatalf("expected job title, got %q", detail.JobTitle)
	}
	if len(detail.Questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(detail.Questions))
	}
	if detail.Questions[0].Order != 1 || detail.Questions[0].Text != "問題一" {
		t.Fatalf("expected first ordered question, got %+v", detail.Questions[0])
	}
	if len(detail.Answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(detail.Answers))
	}
	if detail.Answers[0].QuestionID != firstQuestionID {
		t.Fatalf("expected answer question id %q, got %q", firstQuestionID, detail.Answers[0].QuestionID)
	}
	if detail.Answers[0].AudioPath == nil || *detail.Answers[0].AudioPath != audioPath {
		t.Fatalf("expected audio path %q, got %+v", audioPath, detail.Answers[0].AudioPath)
	}
	if detail.Answers[0].TranscriptText != nil {
		t.Fatalf("expected nil transcript text, got %+v", detail.Answers[0].TranscriptText)
	}
}
