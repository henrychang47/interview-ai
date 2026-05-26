package repository

import (
	"context"

	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InterviewRepository struct {
	pool *pgxpool.Pool
}

func NewInterviewRepository(pool *pgxpool.Pool) *InterviewRepository {
	return &InterviewRepository{pool: pool}
}

func (r *InterviewRepository) CreateWithQuestions(
	ctx context.Context,
	input model.CreateInterviewRequest,
	questions []llm.GeneratedQuestion,
) (model.CreateInterviewResponse, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return model.CreateInterviewResponse{}, err
	}
	defer tx.Rollback(ctx)

	var interviewID string
	err = tx.QueryRow(ctx, `
		INSERT INTO interviews (job_title, job_description, user_profile, question_count, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, input.JobTitle, input.JobDescription, input.UserProfile, input.QuestionCount, model.InterviewStatusQuestionsReady).Scan(&interviewID)
	if err != nil {
		return model.CreateInterviewResponse{}, err
	}

	for _, question := range questions {
		_, err = tx.Exec(ctx, `
			INSERT INTO questions (interview_id, question_order, question_text)
			VALUES ($1, $2, $3)
		`, interviewID, question.Order, question.Text)
		if err != nil {
			return model.CreateInterviewResponse{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return model.CreateInterviewResponse{}, err
	}

	return model.CreateInterviewResponse{
		ID:     interviewID,
		Status: model.InterviewStatusQuestionsReady,
	}, nil
}
