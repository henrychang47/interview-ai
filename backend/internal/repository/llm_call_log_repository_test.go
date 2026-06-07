package repository

import (
	"context"
	"os"
	"testing"

	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCreateLLMCallLogInsertsNullableFields(t *testing.T) {
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
		QuestionCount:  1,
	}, []llm.GeneratedQuestion{{Order: 1, Text: "問題一"}})
	if err != nil {
		t.Fatalf("createInterviewWithQuestions returned error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", created.ID)
	})

	repository := NewLLMCallLogRepository(pool)
	err = repository.CreateLLMCallLog(ctx, model.LLMCallLog{
		Operation:   model.LLMOperationGenerateQuestions,
		Provider:    model.LLMProviderGemini,
		Model:       "gemini-2.5-flash",
		InterviewID: &created.ID,
		Status:      model.LLMCallStatusSuccess,
		LatencyMS:   intPtr(123),
		InputTokens: intPtr(10),
		TotalTokens: intPtr(15),
	})
	if err != nil {
		t.Fatalf("CreateLLMCallLog returned error: %v", err)
	}

	var count int
	var outputTokens *int
	var answerID *string
	if err := pool.QueryRow(ctx, `
		SELECT count(*), max(output_tokens), max(answer_id::text)
		FROM llm_call_logs
		WHERE interview_id = $1
		  AND operation = $2
		  AND provider = $3
		  AND model = $4
		  AND status = $5
		  AND latency_ms = 123
		  AND input_tokens = 10
		  AND total_tokens = 15
	`, created.ID, model.LLMOperationGenerateQuestions, model.LLMProviderGemini, "gemini-2.5-flash", model.LLMCallStatusSuccess).Scan(
		&count,
		&outputTokens,
		&answerID,
	); err != nil {
		t.Fatalf("query inserted LLM call log: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one LLM call log, got %d", count)
	}
	if outputTokens != nil {
		t.Fatalf("expected nil output tokens, got %d", *outputTokens)
	}
	if answerID != nil {
		t.Fatalf("expected nil answer id, got %q", *answerID)
	}
}

func intPtr(value int) *int {
	return &value
}
