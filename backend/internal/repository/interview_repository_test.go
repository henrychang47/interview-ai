package repository

import (
	"context"
	"errors"
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

func TestCreatePendingPersistsGeneratingInterviewWithLanguage(t *testing.T) {
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
	response, err := repository.CreatePending(ctx, model.CreateInterviewRequest{
		JobTitle:         "Backend Engineer",
		JobDescription:   "Build APIs",
		UserProfile:      "Go experience",
		QuestionCount:    2,
		QuestionLanguage: model.QuestionLanguageEnUS,
	})
	if err != nil {
		t.Fatalf("CreatePending returned error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", response.ID)
	})

	if response.Status != model.InterviewStatusGeneratingQuestions {
		t.Fatalf("expected generating_questions, got %q", response.Status)
	}

	var status, language string
	if err := pool.QueryRow(ctx, `
		SELECT status, question_language
		FROM interviews
		WHERE id = $1
	`, response.ID).Scan(&status, &language); err != nil {
		t.Fatalf("query interview: %v", err)
	}
	if status != model.InterviewStatusGeneratingQuestions || language != model.QuestionLanguageEnUS {
		t.Fatalf("unexpected status/language: %q/%q", status, language)
	}
}

func TestSaveGeneratedQuestionsMarksInterviewReady(t *testing.T) {
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
	created, err := repository.CreatePending(ctx, model.CreateInterviewRequest{
		JobTitle:         "後端工程師",
		JobDescription:   "需要熟悉 Go",
		UserProfile:      "有 Go 經驗",
		QuestionCount:    2,
		QuestionLanguage: model.QuestionLanguageZhTW,
	})
	if err != nil {
		t.Fatalf("CreatePending returned error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", created.ID)
	})

	err = repository.SaveGeneratedQuestions(ctx, created.ID, []llm.GeneratedQuestion{
		{Order: 1, Text: "問題一"},
		{Order: 2, Text: "問題二"},
	})
	if err != nil {
		t.Fatalf("SaveGeneratedQuestions returned error: %v", err)
	}

	var status string
	var questionCount int
	if err := pool.QueryRow(ctx, `
		SELECT status, (SELECT count(*) FROM questions WHERE interview_id = $1)
		FROM interviews
		WHERE id = $1
	`, created.ID).Scan(&status, &questionCount); err != nil {
		t.Fatalf("query generated status: %v", err)
	}
	if status != model.InterviewStatusQuestionsReady || questionCount != 2 {
		t.Fatalf("expected ready with 2 questions, got %q with %d", status, questionCount)
	}
}

func TestMarkQuestionGenerationFailedMarksInterviewFailed(t *testing.T) {
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
	created, err := repository.CreatePending(ctx, model.CreateInterviewRequest{
		JobTitle:         "後端工程師",
		JobDescription:   "需要熟悉 Go",
		UserProfile:      "有 Go 經驗",
		QuestionCount:    1,
		QuestionLanguage: model.QuestionLanguageZhTW,
	})
	if err != nil {
		t.Fatalf("CreatePending returned error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", created.ID)
	})

	if err := repository.MarkQuestionGenerationFailed(ctx, created.ID); err != nil {
		t.Fatalf("MarkQuestionGenerationFailed returned error: %v", err)
	}

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM interviews WHERE id = $1`, created.ID).Scan(&status); err != nil {
		t.Fatalf("query status: %v", err)
	}
	if status != model.InterviewStatusFailed {
		t.Fatalf("expected failed, got %q", status)
	}
}

func TestStartMovesReadyInterviewToInProgress(t *testing.T) {
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
	created, err := repository.CreatePending(ctx, model.CreateInterviewRequest{
		JobTitle:         "後端工程師",
		JobDescription:   "需要熟悉 Go",
		UserProfile:      "有 Go 經驗",
		QuestionCount:    1,
		QuestionLanguage: model.QuestionLanguageZhTW,
	})
	if err != nil {
		t.Fatalf("CreatePending returned error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", created.ID)
	})

	if err := repository.SaveGeneratedQuestions(ctx, created.ID, []llm.GeneratedQuestion{{Order: 1, Text: "問題一"}}); err != nil {
		t.Fatalf("SaveGeneratedQuestions returned error: %v", err)
	}

	response, err := repository.Start(ctx, created.ID)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if response.Status != model.InterviewStatusInProgress {
		t.Fatalf("expected in_progress, got %q", response.Status)
	}
}

func TestStartRejectsInterviewThatIsNotReady(t *testing.T) {
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
	created, err := repository.CreatePending(ctx, model.CreateInterviewRequest{
		JobTitle:         "後端工程師",
		JobDescription:   "需要熟悉 Go",
		UserProfile:      "有 Go 經驗",
		QuestionCount:    1,
		QuestionLanguage: model.QuestionLanguageZhTW,
	})
	if err != nil {
		t.Fatalf("CreatePending returned error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", created.ID)
	})

	_, err = repository.Start(ctx, created.ID)
	if !errors.Is(err, ErrInterviewNotReady) {
		t.Fatalf("expected ErrInterviewNotReady, got %v", err)
	}
}

func TestStartReturnsNotFoundForMissingInterview(t *testing.T) {
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

	_, err = repository.Start(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, model.ErrInterviewNotFound) {
		t.Fatalf("expected ErrInterviewNotFound, got %v", err)
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
