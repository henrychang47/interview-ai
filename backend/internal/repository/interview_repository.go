package repository

import (
	"context"
	"errors"
	"time"

	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InterviewRepository struct {
	pool *pgxpool.Pool
}

var ErrInterviewNotReady = model.ErrInterviewNotReady

func NewInterviewRepository(pool *pgxpool.Pool) *InterviewRepository {
	return &InterviewRepository{pool: pool}
}

func (r *InterviewRepository) CreatePending(ctx context.Context, input model.CreateInterviewRequest) (model.CreateInterviewResponse, error) {
	return createPending(ctx, r.pool, input)
}

func (r *InterviewRepository) CreatePendingWithCreationLimit(ctx context.Context, input model.CreateInterviewRequest, clientIPHash string, limit int) (model.CreateInterviewResponse, error) {
	if clientIPHash == "" {
		return r.CreatePending(ctx, input)
	}
	if limit <= 0 {
		limit = 5
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return model.CreateInterviewResponse{}, err
	}
	defer tx.Rollback(ctx)

	if err := reserveInterviewCreation(ctx, tx, clientIPHash, limit); err != nil {
		return model.CreateInterviewResponse{}, err
	}

	created, err := createPending(ctx, tx, input)
	if err != nil {
		return model.CreateInterviewResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.CreateInterviewResponse{}, err
	}

	return created, nil
}

type interviewCreator interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func createPending(ctx context.Context, creator interviewCreator, input model.CreateInterviewRequest) (model.CreateInterviewResponse, error) {
	var interviewID string
	err := creator.QueryRow(ctx, `
		INSERT INTO interviews (job_title, job_description, user_profile, question_count, question_language, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, input.JobTitle, input.JobDescription, input.UserProfile, input.QuestionCount, input.QuestionLanguage, model.InterviewStatusGeneratingQuestions).Scan(&interviewID)
	if err != nil {
		return model.CreateInterviewResponse{}, err
	}

	return model.CreateInterviewResponse{
		ID:     interviewID,
		Status: model.InterviewStatusGeneratingQuestions,
	}, nil
}

func reserveInterviewCreation(ctx context.Context, tx pgx.Tx, clientIPHash string, limit int) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO interview_creation_limits (ip_hash, window_started_at, created_count, updated_at)
		VALUES ($1, now(), 0, now())
		ON CONFLICT (ip_hash) DO NOTHING
	`, clientIPHash)
	if err != nil {
		return err
	}

	var windowStartedAt time.Time
	var createdCount int
	var currentTime time.Time
	if err := tx.QueryRow(ctx, `
		SELECT window_started_at, created_count, now()
		FROM interview_creation_limits
		WHERE ip_hash = $1
		FOR UPDATE
	`, clientIPHash).Scan(&windowStartedAt, &createdCount, &currentTime); err != nil {
		return err
	}

	if !currentTime.Before(windowStartedAt.Add(24 * time.Hour)) {
		_, err = tx.Exec(ctx, `
			UPDATE interview_creation_limits
			SET window_started_at = $2, created_count = 1, updated_at = $2
			WHERE ip_hash = $1
		`, clientIPHash, currentTime)
		return err
	}

	if createdCount >= limit {
		return model.ErrInterviewCreationLimitReached
	}

	_, err = tx.Exec(ctx, `
		UPDATE interview_creation_limits
		SET created_count = created_count + 1, updated_at = $2
		WHERE ip_hash = $1
	`, clientIPHash, currentTime)
	return err
}

func (r *InterviewRepository) SaveGeneratedQuestions(ctx context.Context, interviewID string, questions []llm.GeneratedQuestion) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		DELETE FROM questions
		WHERE interview_id = $1
	`, interviewID); err != nil {
		return err
	}

	for _, question := range questions {
		_, err = tx.Exec(ctx, `
			INSERT INTO questions (interview_id, question_order, question_text)
			VALUES ($1, $2, $3)
		`, interviewID, question.Order, question.Text)
		if err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE interviews
		SET status = $2, updated_at = now()
		WHERE id = $1
	`, interviewID, model.InterviewStatusQuestionsReady); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (r *InterviewRepository) MarkQuestionGenerationFailed(ctx context.Context, interviewID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE interviews
		SET status = $2, updated_at = now()
		WHERE id = $1
	`, interviewID, model.InterviewStatusFailed)
	return err
}

func (r *InterviewRepository) Start(ctx context.Context, interviewID string) (model.CreateInterviewResponse, error) {
	var response model.CreateInterviewResponse
	err := r.pool.QueryRow(ctx, `
		UPDATE interviews
		SET status = $2, updated_at = now()
		WHERE id = $1 AND status = $3
		RETURNING id, status
	`, interviewID, model.InterviewStatusInProgress, model.InterviewStatusQuestionsReady).Scan(&response.ID, &response.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			var exists bool
			if existsErr := r.pool.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1
					FROM interviews
					WHERE id = $1
				)
			`, interviewID).Scan(&exists); existsErr != nil {
				return model.CreateInterviewResponse{}, existsErr
			}
			if !exists {
				return model.CreateInterviewResponse{}, model.ErrInterviewNotFound
			}
			return model.CreateInterviewResponse{}, ErrInterviewNotReady
		}
		return model.CreateInterviewResponse{}, err
	}
	return response, nil
}

func (r *InterviewRepository) GetByID(ctx context.Context, interviewID string) (model.InterviewDetail, error) {
	var detail model.InterviewDetail
	err := r.pool.QueryRow(ctx, `
		SELECT id, job_title, job_description, user_profile, question_count, question_language, status
		FROM interviews
		WHERE id = $1
	`, interviewID).Scan(
		&detail.ID,
		&detail.JobTitle,
		&detail.JobDescription,
		&detail.UserProfile,
		&detail.QuestionCount,
		&detail.QuestionLanguage,
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
		SELECT
			id,
			interview_id,
			question_id,
			audio_path,
			transcript_text,
			analysis_status,
			improvement_suggestions,
			analysis_error,
			analyzed_at,
			created_at
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
			&answer.AnalysisStatus,
			&answer.ImprovementSuggestions,
			&answer.AnalysisError,
			&answer.AnalyzedAt,
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
