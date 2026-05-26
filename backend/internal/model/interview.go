package model

import "time"

const (
	InterviewStatusCreated        = "created"
	InterviewStatusQuestionsReady = "questions_ready"
)

type CreateInterviewRequest struct {
	JobTitle       string `json:"job_title"`
	JobDescription string `json:"job_description"`
	UserProfile    string `json:"user_profile"`
	QuestionCount  int    `json:"question_count"`
}

type CreateInterviewResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type Interview struct {
	ID             string
	JobTitle       string
	JobDescription string
	UserProfile    string
	QuestionCount  int
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Question struct {
	ID          string
	InterviewID string
	Order       int
	Text        string
	CreatedAt   time.Time
}
