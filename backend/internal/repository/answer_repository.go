package repository

import (
	"context"

	"interview-ai/backend/internal/model"
	"interview-ai/backend/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AnswerRepository struct {
	pool *pgxpool.Pool
}

func NewAnswerRepository(pool *pgxpool.Pool) *AnswerRepository {
	return &AnswerRepository{pool: pool}
}

func (r *AnswerRepository) EnsureQuestionForInterview(ctx context.Context, interviewID string, questionID string) error {
	var interviewExists bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM interviews
			WHERE id = $1
		)
	`, interviewID).Scan(&interviewExists); err != nil {
		return err
	}
	if !interviewExists {
		return service.ErrInterviewNotFound
	}

	var questionExists bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM questions
			WHERE id = $1 AND interview_id = $2
		)
	`, questionID, interviewID).Scan(&questionExists); err != nil {
		return err
	}
	if !questionExists {
		return service.ErrQuestionNotFoundForInterview
	}

	return nil
}

func (r *AnswerRepository) UpsertAnswer(ctx context.Context, interviewID string, questionID string, audioPath string) (model.Answer, error) {
	if err := r.EnsureQuestionForInterview(ctx, interviewID, questionID); err != nil {
		return model.Answer{}, err
	}

	var answer model.Answer
	if err := r.pool.QueryRow(ctx, `
		INSERT INTO answers (interview_id, question_id, audio_path, transcript_text)
		VALUES ($1, $2, $3, NULL)
		ON CONFLICT (interview_id, question_id)
		DO UPDATE SET audio_path = EXCLUDED.audio_path
		RETURNING id, interview_id, question_id, audio_path, transcript_text, created_at
	`, interviewID, questionID, audioPath).Scan(
		&answer.ID,
		&answer.InterviewID,
		&answer.QuestionID,
		&answer.AudioPath,
		&answer.TranscriptText,
		&answer.CreatedAt,
	); err != nil {
		return model.Answer{}, err
	}

	return answer, nil
}
