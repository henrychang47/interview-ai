package repository

import (
	"context"
	"time"

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
		INSERT INTO answers (
			interview_id,
			question_id,
			audio_path,
			transcript_text,
			analysis_status,
			improvement_suggestions,
			analysis_error,
			analyzed_at
		)
		VALUES ($1, $2, $3, NULL, $4, NULL, NULL, NULL)
		ON CONFLICT (interview_id, question_id)
		DO UPDATE SET
			audio_path = EXCLUDED.audio_path,
			transcript_text = NULL,
			analysis_status = EXCLUDED.analysis_status,
			improvement_suggestions = NULL,
			analysis_error = NULL,
			analyzed_at = NULL,
			created_at = now()
		RETURNING
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
	`, interviewID, questionID, audioPath, model.AnswerAnalysisStatusPending).Scan(
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
		return model.Answer{}, err
	}

	return answer, nil
}

func (r *AnswerRepository) ListExpiredAnswerAudio(ctx context.Context, cutoff time.Time) ([]model.ExpiredAnswerAudio, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, audio_path
		FROM answers
		WHERE audio_path IS NOT NULL
		  AND created_at < $1
		ORDER BY created_at, id
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expiredAudio := []model.ExpiredAnswerAudio{}
	for rows.Next() {
		var audio model.ExpiredAnswerAudio
		if err := rows.Scan(&audio.AnswerID, &audio.AudioPath); err != nil {
			return nil, err
		}
		expiredAudio = append(expiredAudio, audio)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return expiredAudio, nil
}

func (r *AnswerRepository) ClearAnswerAudioPath(ctx context.Context, answerID string, audioPath string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE answers
		SET audio_path = NULL
		WHERE id = $1
		  AND audio_path = $2
	`, answerID, audioPath)
	return err
}

func (r *AnswerRepository) GetAnswerAnalysisContext(ctx context.Context, interviewID string, questionID string) (model.AnswerAnalysisContext, error) {
	if err := r.EnsureQuestionForInterview(ctx, interviewID, questionID); err != nil {
		return model.AnswerAnalysisContext{}, err
	}

	var analysisContext model.AnswerAnalysisContext
	if err := r.pool.QueryRow(ctx, `
		SELECT
			i.job_title,
			i.job_description,
			i.user_profile,
			q.question_text
		FROM interviews i
		JOIN questions q ON q.interview_id = i.id
		WHERE i.id = $1 AND q.id = $2
	`, interviewID, questionID).Scan(
		&analysisContext.JobTitle,
		&analysisContext.JobDescription,
		&analysisContext.UserProfile,
		&analysisContext.QuestionText,
	); err != nil {
		return model.AnswerAnalysisContext{}, err
	}

	return analysisContext, nil
}

func (r *AnswerRepository) MarkAnswerAnalysisProcessing(ctx context.Context, answerID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE answers
		SET analysis_status = $2,
		    analysis_error = NULL,
		    analyzed_at = NULL
		WHERE id = $1
	`, answerID, model.AnswerAnalysisStatusProcessing)
	return err
}

func (r *AnswerRepository) MarkAnswerAnalysisCompleted(ctx context.Context, answerID string, transcriptText string, improvementSuggestions string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE answers
		SET transcript_text = $2,
		    improvement_suggestions = $3,
		    analysis_status = $4,
		    analysis_error = NULL,
		    analyzed_at = now()
		WHERE id = $1
	`, answerID, transcriptText, improvementSuggestions, model.AnswerAnalysisStatusCompleted)
	return err
}

func (r *AnswerRepository) MarkAnswerAnalysisFailed(ctx context.Context, answerID string, message string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE answers
		SET analysis_status = $2,
		    analysis_error = $3,
		    analyzed_at = NULL
		WHERE id = $1
	`, answerID, model.AnswerAnalysisStatusFailed, message)
	return err
}

func (r *AnswerRepository) CompleteInterviewIfAllQuestionsAnswered(ctx context.Context, interviewID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE interviews
		SET status = $2, updated_at = now()
		WHERE id = $1
		  AND (
			SELECT count(*)
			FROM questions
			WHERE interview_id = $1
		  ) > 0
		  AND (
			SELECT count(DISTINCT question_id)
			FROM answers
			WHERE interview_id = $1
		  ) = (
			SELECT count(*)
			FROM questions
			WHERE interview_id = $1
		  )
	`, interviewID, model.InterviewStatusCompleted)
	return err
}
