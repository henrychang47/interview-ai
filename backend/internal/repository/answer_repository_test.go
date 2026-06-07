package repository

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

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
	t.Cleanup(func() {
		pool.Close()
	})

	interviewRepository := NewInterviewRepository(pool)
	created, err := createInterviewWithQuestions(ctx, interviewRepository, model.CreateInterviewRequest{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go、PostgreSQL、REST API",
		UserProfile:    "有 Go 學習經驗",
		QuestionCount:  1,
	}, []llm.GeneratedQuestion{{Order: 1, Text: "問題一"}})
	if err != nil {
		t.Fatalf("createInterviewWithQuestions returned error: %v", err)
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

	oldCreatedAt := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `UPDATE answers SET created_at = $2 WHERE id = $1`, answer.ID, oldCreatedAt); err != nil {
		t.Fatalf("set old created_at: %v", err)
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
	if !updated.CreatedAt.After(oldCreatedAt) {
		t.Fatalf("expected re-upload to refresh created_at after %v, got %v", oldCreatedAt, updated.CreatedAt)
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

func TestListExpiredAnswerAudioAndClearAnswerAudioPath(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})

	interviewRepository := NewInterviewRepository(pool)
	created, err := createInterviewWithQuestions(ctx, interviewRepository, model.CreateInterviewRequest{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go",
		UserProfile:    "有 Go 學習經驗",
		QuestionCount:  3,
	}, []llm.GeneratedQuestion{
		{Order: 1, Text: "第一題"},
		{Order: 2, Text: "第二題"},
		{Order: 3, Text: "第三題"},
	})
	if err != nil {
		t.Fatalf("createInterviewWithQuestions returned error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", created.ID)
	})

	rows, err := pool.Query(ctx, `
		SELECT id
		FROM questions
		WHERE interview_id = $1
		ORDER BY question_order
	`, created.ID)
	if err != nil {
		t.Fatalf("query questions: %v", err)
	}
	defer rows.Close()

	questionIDs := make([]string, 0, 3)
	for rows.Next() {
		var questionID string
		if err := rows.Scan(&questionID); err != nil {
			t.Fatalf("scan question id: %v", err)
		}
		questionIDs = append(questionIDs, questionID)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate questions: %v", err)
	}
	if len(questionIDs) != 3 {
		t.Fatalf("expected 3 questions, got %d", len(questionIDs))
	}

	repository := NewAnswerRepository(pool)
	expiredAnswer, err := repository.UpsertAnswer(ctx, created.ID, questionIDs[0], "storage/audio/expired.webm")
	if err != nil {
		t.Fatalf("UpsertAnswer expired returned error: %v", err)
	}
	_, err = repository.UpsertAnswer(ctx, created.ID, questionIDs[1], "storage/audio/fresh.webm")
	if err != nil {
		t.Fatalf("UpsertAnswer fresh returned error: %v", err)
	}
	nullPathAnswer, err := repository.UpsertAnswer(ctx, created.ID, questionIDs[2], "storage/audio/null-path.webm")
	if err != nil {
		t.Fatalf("UpsertAnswer null-path returned error: %v", err)
	}

	expiredCreatedAt := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	freshCreatedAt := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `UPDATE answers SET created_at = $2 WHERE id = $1`, expiredAnswer.ID, expiredCreatedAt); err != nil {
		t.Fatalf("set expired created_at: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE answers SET created_at = $2 WHERE question_id = $1`, questionIDs[1], freshCreatedAt); err != nil {
		t.Fatalf("set fresh created_at: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE answers SET created_at = $2, audio_path = NULL WHERE id = $1`, nullPathAnswer.ID, expiredCreatedAt); err != nil {
		t.Fatalf("set null-path answer: %v", err)
	}

	expired, err := repository.ListExpiredAnswerAudio(ctx, time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ListExpiredAnswerAudio returned error: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("expected one expired answer audio, got %+v", expired)
	}
	if expired[0].AnswerID != expiredAnswer.ID {
		t.Fatalf("expected expired answer id %q, got %q", expiredAnswer.ID, expired[0].AnswerID)
	}
	if expired[0].AudioPath != "storage/audio/expired.webm" {
		t.Fatalf("expected expired audio path, got %q", expired[0].AudioPath)
	}

	if err := repository.ClearAnswerAudioPath(ctx, expiredAnswer.ID, "storage/audio/wrong.webm"); err != nil {
		t.Fatalf("ClearAnswerAudioPath wrong path returned error: %v", err)
	}
	var stillPresent *string
	if err := pool.QueryRow(ctx, `SELECT audio_path FROM answers WHERE id = $1`, expiredAnswer.ID).Scan(&stillPresent); err != nil {
		t.Fatalf("query audio path after wrong clear: %v", err)
	}
	if stillPresent == nil || *stillPresent != "storage/audio/expired.webm" {
		t.Fatalf("expected guarded clear to preserve audio path, got %+v", stillPresent)
	}

	if err := repository.ClearAnswerAudioPath(ctx, expiredAnswer.ID, "storage/audio/expired.webm"); err != nil {
		t.Fatalf("ClearAnswerAudioPath returned error: %v", err)
	}
	var clearedPath *string
	if err := pool.QueryRow(ctx, `SELECT audio_path FROM answers WHERE id = $1`, expiredAnswer.ID).Scan(&clearedPath); err != nil {
		t.Fatalf("query audio path after clear: %v", err)
	}
	if clearedPath != nil {
		t.Fatalf("expected cleared audio path, got %+v", clearedPath)
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
	t.Cleanup(func() {
		pool.Close()
	})

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
	t.Cleanup(func() {
		pool.Close()
	})

	interviewRepository := NewInterviewRepository(pool)
	first, err := createInterviewWithQuestions(ctx, interviewRepository, model.CreateInterviewRequest{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go",
		UserProfile:    "有 Go 學習經驗",
		QuestionCount:  1,
	}, []llm.GeneratedQuestion{{Order: 1, Text: "第一場問題"}})
	if err != nil {
		t.Fatalf("create first interview: %v", err)
	}
	second, err := createInterviewWithQuestions(ctx, interviewRepository, model.CreateInterviewRequest{
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

func TestGetAnswerAnalysisContextReturnsInterviewAndQuestionContext(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})

	interviewRepository := NewInterviewRepository(pool)
	created, err := createInterviewWithQuestions(ctx, interviewRepository, model.CreateInterviewRequest{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go、PostgreSQL、REST API",
		UserProfile:    "有 Java 和 Go 學習經驗",
		QuestionCount:  1,
	}, []llm.GeneratedQuestion{{Order: 1, Text: "請介紹你做過的 Go API 專案。"}})
	if err != nil {
		t.Fatalf("createInterviewWithQuestions returned error: %v", err)
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
	analysisContext, err := repository.GetAnswerAnalysisContext(ctx, created.ID, questionID)
	if err != nil {
		t.Fatalf("GetAnswerAnalysisContext returned error: %v", err)
	}

	if analysisContext.JobTitle != "後端工程師" {
		t.Fatalf("expected job title, got %q", analysisContext.JobTitle)
	}
	if analysisContext.JobDescription != "需要熟悉 Go、PostgreSQL、REST API" {
		t.Fatalf("expected job description, got %q", analysisContext.JobDescription)
	}
	if analysisContext.UserProfile != "有 Java 和 Go 學習經驗" {
		t.Fatalf("expected user profile, got %q", analysisContext.UserProfile)
	}
	if analysisContext.QuestionText != "請介紹你做過的 Go API 專案。" {
		t.Fatalf("expected question text, got %q", analysisContext.QuestionText)
	}
}

func TestGetAnswerAnalysisContextRejectsQuestionOutsideInterview(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})

	interviewRepository := NewInterviewRepository(pool)
	first, err := createInterviewWithQuestions(ctx, interviewRepository, model.CreateInterviewRequest{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go",
		UserProfile:    "有 Go 學習經驗",
		QuestionCount:  1,
	}, []llm.GeneratedQuestion{{Order: 1, Text: "第一場問題"}})
	if err != nil {
		t.Fatalf("create first interview: %v", err)
	}
	second, err := createInterviewWithQuestions(ctx, interviewRepository, model.CreateInterviewRequest{
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
	_, err = repository.GetAnswerAnalysisContext(ctx, first.ID, secondQuestionID)

	if !errors.Is(err, service.ErrQuestionNotFoundForInterview) {
		t.Fatalf("expected ErrQuestionNotFoundForInterview, got %v", err)
	}
}

func TestCompleteInterviewIfAllQuestionsAnsweredKeepsInterviewOpenBeforeFinalAnswer(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})

	interviewRepository := NewInterviewRepository(pool)
	created, err := createInterviewWithQuestions(ctx, interviewRepository, model.CreateInterviewRequest{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go",
		UserProfile:    "有 Go 學習經驗",
		QuestionCount:  2,
	}, []llm.GeneratedQuestion{
		{Order: 1, Text: "第一題"},
		{Order: 2, Text: "第二題"},
	})
	if err != nil {
		t.Fatalf("createInterviewWithQuestions returned error: %v", err)
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

	repository := NewAnswerRepository(pool)
	if _, err := repository.UpsertAnswer(ctx, created.ID, firstQuestionID, "storage/audio/first.webm"); err != nil {
		t.Fatalf("UpsertAnswer returned error: %v", err)
	}
	if err := repository.CompleteInterviewIfAllQuestionsAnswered(ctx, created.ID); err != nil {
		t.Fatalf("CompleteInterviewIfAllQuestionsAnswered returned error: %v", err)
	}

	var status string
	if err := pool.QueryRow(ctx, `
		SELECT status
		FROM interviews
		WHERE id = $1
	`, created.ID).Scan(&status); err != nil {
		t.Fatalf("query interview status: %v", err)
	}
	if status != model.InterviewStatusQuestionsReady {
		t.Fatalf("expected status %q, got %q", model.InterviewStatusQuestionsReady, status)
	}
}

func TestCompleteInterviewIfAllQuestionsAnsweredMarksInterviewCompleted(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})

	interviewRepository := NewInterviewRepository(pool)
	created, err := createInterviewWithQuestions(ctx, interviewRepository, model.CreateInterviewRequest{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go",
		UserProfile:    "有 Go 學習經驗",
		QuestionCount:  2,
	}, []llm.GeneratedQuestion{
		{Order: 1, Text: "第一題"},
		{Order: 2, Text: "第二題"},
	})
	if err != nil {
		t.Fatalf("createInterviewWithQuestions returned error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", created.ID)
	})

	rows, err := pool.Query(ctx, `
		SELECT id
		FROM questions
		WHERE interview_id = $1
		ORDER BY question_order
	`, created.ID)
	if err != nil {
		t.Fatalf("query questions: %v", err)
	}
	defer rows.Close()

	questionIDs := make([]string, 0, 2)
	for rows.Next() {
		var questionID string
		if err := rows.Scan(&questionID); err != nil {
			t.Fatalf("scan question id: %v", err)
		}
		questionIDs = append(questionIDs, questionID)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate questions: %v", err)
	}
	if len(questionIDs) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(questionIDs))
	}

	repository := NewAnswerRepository(pool)
	for index, questionID := range questionIDs {
		if _, err := repository.UpsertAnswer(ctx, created.ID, questionID, "storage/audio/answer-"+strconv.Itoa(index+1)+".webm"); err != nil {
			t.Fatalf("UpsertAnswer question %d returned error: %v", index+1, err)
		}
	}
	if err := repository.CompleteInterviewIfAllQuestionsAnswered(ctx, created.ID); err != nil {
		t.Fatalf("CompleteInterviewIfAllQuestionsAnswered returned error: %v", err)
	}

	var status string
	if err := pool.QueryRow(ctx, `
		SELECT status
		FROM interviews
		WHERE id = $1
	`, created.ID).Scan(&status); err != nil {
		t.Fatalf("query interview status: %v", err)
	}
	if status != model.InterviewStatusCompleted {
		t.Fatalf("expected status %q, got %q", model.InterviewStatusCompleted, status)
	}
}
