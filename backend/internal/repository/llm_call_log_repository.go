package repository

import (
	"context"

	"interview-ai/backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LLMCallLogRepository struct {
	pool *pgxpool.Pool
}

func NewLLMCallLogRepository(pool *pgxpool.Pool) *LLMCallLogRepository {
	return &LLMCallLogRepository{pool: pool}
}

func (r *LLMCallLogRepository) CreateLLMCallLog(ctx context.Context, log model.LLMCallLog) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO llm_call_logs (
			operation,
			provider,
			model,
			interview_id,
			question_id,
			answer_id,
			status,
			latency_ms,
			input_tokens,
			output_tokens,
			total_tokens,
			error_code,
			error_message
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, log.Operation,
		log.Provider,
		log.Model,
		log.InterviewID,
		log.QuestionID,
		log.AnswerID,
		log.Status,
		log.LatencyMS,
		log.InputTokens,
		log.OutputTokens,
		log.TotalTokens,
		log.ErrorCode,
		log.ErrorMessage,
	)
	return err
}
