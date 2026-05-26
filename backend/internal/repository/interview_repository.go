package repository

import (
	"context"
	"errors"

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

func (r *InterviewRepository) GetByID(ctx context.Context, interviewID string) (model.InterviewDetail, error) {
	var detail model.InterviewDetail
	err := r.pool.QueryRow(ctx, `
		SELECT id, job_title, job_description, user_profile, question_count, status
		FROM interviews
		WHERE id = $1
	`, interviewID).Scan(
		&detail.ID,
		&detail.JobTitle,
		&detail.JobDescription,
		&detail.UserProfile,
		&detail.QuestionCount,
		&detail.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.InterviewDetail{}, pgx.ErrNoRows
		}
		return model.InterviewDetail{}, err
	}

	questionRows, err := r.pool.Query(ctx, `
		SELECT id, interview_id, question_order, question_text, created_at
		FROM questions
		WHERE interview_id = $1
		ORDER BY question_order
	`, interviewID)
	if err != nil {
		return model.InterviewDetail{}, err
	}
	defer questionRows.Close()

	questions := make([]model.Question, 0)
	for questionRows.Next() {
		var question model.Question
		if err := questionRows.Scan(
			&question.ID,
			&question.InterviewID,
			&question.Order,
			&question.Text,
			&question.CreatedAt,
		); err != nil {
			return model.InterviewDetail{}, err
		}
		questions = append(questions, question)
	}
	if err := questionRows.Err(); err != nil {
		return model.InterviewDetail{}, err
	}

	answerRows, err := r.pool.Query(ctx, `
		SELECT id, interview_id, question_id, audio_path, transcript_text, created_at
		FROM answers
		WHERE interview_id = $1
		ORDER BY created_at
	`, interviewID)
	if err != nil {
		return model.InterviewDetail{}, err
	}
	defer answerRows.Close()

	answers := make([]model.Answer, 0)
	for answerRows.Next() {
		var answer model.Answer
		if err := answerRows.Scan(
			&answer.ID,
			&answer.InterviewID,
			&answer.QuestionID,
			&answer.AudioPath,
			&answer.TranscriptText,
			&answer.CreatedAt,
		); err != nil {
			return model.InterviewDetail{}, err
		}
		answers = append(answers, answer)
	}
	if err := answerRows.Err(); err != nil {
		return model.InterviewDetail{}, err
	}

	detail.Questions = questions
	detail.Answers = answers
	return detail, nil
}
