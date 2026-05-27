package model

import (
	"errors"
	"time"
)

var ErrInterviewNotReady = errors.New("interview is not ready to start")

const (
	InterviewStatusCreated             = "created"
	InterviewStatusGeneratingQuestions = "generating_questions"
	InterviewStatusQuestionsReady      = "questions_ready"
	InterviewStatusInProgress          = "in_progress"
	InterviewStatusCompleted           = "completed"
	InterviewStatusFailed              = "failed"

	QuestionLanguageZhTW = "zh-TW"
	QuestionLanguageEnUS = "en-US"
)

type CreateInterviewRequest struct {
	JobTitle         string `json:"job_title"`
	JobDescription   string `json:"job_description"`
	UserProfile      string `json:"user_profile"`
	QuestionCount    int    `json:"question_count"`
	QuestionLanguage string `json:"question_language"`
}

type CreateInterviewResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type Interview struct {
	ID               string
	JobTitle         string
	JobDescription   string
	UserProfile      string
	QuestionCount    int
	QuestionLanguage string
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Question struct {
	ID          string
	InterviewID string
	Order       int
	Text        string
	CreatedAt   time.Time
}

type InterviewDetail struct {
	ID               string
	JobTitle         string
	JobDescription   string
	UserProfile      string
	QuestionCount    int
	QuestionLanguage string
	Status           string
	Questions        []Question
	Answers          []Answer
}

type Answer struct {
	ID             string
	InterviewID    string
	QuestionID     string
	AudioPath      *string
	TranscriptText *string
	CreatedAt      time.Time
}

type InterviewDetailResponse struct {
	ID               string             `json:"id"`
	JobTitle         string             `json:"job_title"`
	JobDescription   string             `json:"job_description"`
	UserProfile      string             `json:"user_profile"`
	QuestionCount    int                `json:"question_count"`
	QuestionLanguage string             `json:"question_language"`
	Status           string             `json:"status"`
	Questions        []QuestionResponse `json:"questions"`
	Answers          []AnswerResponse   `json:"answers"`
}

type QuestionResponse struct {
	ID    string `json:"id"`
	Order int    `json:"order"`
	Text  string `json:"text"`
}

type AnswerResponse struct {
	ID             string  `json:"id"`
	QuestionID     string  `json:"question_id"`
	AudioPath      *string `json:"audio_path"`
	TranscriptText *string `json:"transcript_text"`
	CreatedAt      string  `json:"created_at"`
}

type UploadAnswerResponse struct {
	ID             string  `json:"id"`
	InterviewID    string  `json:"interview_id"`
	QuestionID     string  `json:"question_id"`
	AudioPath      string  `json:"audio_path"`
	TranscriptText *string `json:"transcript_text"`
}
