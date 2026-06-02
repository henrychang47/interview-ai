package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"interview-ai/backend/internal/model"
)

func TestUploadAnswerSavesAudioAndPersistsAnswer(t *testing.T) {
	storage := &stubAudioStorage{}
	repository := &stubAnswerRepository{
		answer: model.Answer{
			ID:          "answer-id",
			InterviewID: "interview-id",
			QuestionID:  "question-id",
			AudioPath:   stringPointer("storage/audio/interview-id/question-id.webm"),
		},
	}
	service := NewAnswerService(storage, repository)

	response, err := service.UploadAnswer(context.Background(), UploadAnswerInput{
		InterviewID: "interview-id",
		QuestionID:  "question-id",
		ContentType: "audio/webm",
		Audio:       strings.NewReader("webm-bytes"),
	})

	if err != nil {
		t.Fatalf("UploadAnswer returned error: %v", err)
	}
	if storage.interviewID != "interview-id" {
		t.Fatalf("expected storage interview id, got %q", storage.interviewID)
	}
	if storage.questionID != "question-id" {
		t.Fatalf("expected storage question id, got %q", storage.questionID)
	}
	if storage.audioBytes != "webm-bytes" {
		t.Fatalf("expected storage bytes, got %q", storage.audioBytes)
	}
	if repository.audioPath != "storage/audio/interview-id/question-id.webm" {
		t.Fatalf("expected repository audio path, got %q", repository.audioPath)
	}
	if response.ID != "answer-id" {
		t.Fatalf("expected answer id, got %q", response.ID)
	}
	if response.InterviewID != "interview-id" {
		t.Fatalf("expected interview id, got %q", response.InterviewID)
	}
	if response.QuestionID != "question-id" {
		t.Fatalf("expected question id, got %q", response.QuestionID)
	}
	if response.AudioPath != "storage/audio/interview-id/question-id.webm" {
		t.Fatalf("expected response audio path, got %q", response.AudioPath)
	}
	if response.TranscriptText != nil {
		t.Fatalf("expected nil transcript text, got %+v", response.TranscriptText)
	}
	if response.AnalysisStatus != model.AnswerAnalysisStatusPending {
		t.Fatalf("expected pending analysis status, got %q", response.AnalysisStatus)
	}
}

func TestUploadAnswerEnqueuesAnalysisAfterSavingAnswer(t *testing.T) {
	repository := &stubAnswerRepository{
		answer: model.Answer{
			ID:             "answer-id",
			InterviewID:    "interview-id",
			QuestionID:     "question-id",
			AudioPath:      stringPointer("storage/audio/interview-id/question-id.webm"),
			AnalysisStatus: model.AnswerAnalysisStatusPending,
		},
	}
	queue := &stubAnswerAnalysisQueue{}
	service := NewAnswerService(&stubAudioStorage{}, repository, queue)

	_, err := service.UploadAnswer(context.Background(), UploadAnswerInput{
		InterviewID: "interview-id",
		QuestionID:  "question-id",
		ContentType: "audio/webm",
		Audio:       strings.NewReader("webm-bytes"),
	})

	if err != nil {
		t.Fatalf("UploadAnswer returned error: %v", err)
	}
	if len(queue.jobs) != 1 {
		t.Fatalf("expected one analysis job, got %d", len(queue.jobs))
	}
	job := queue.jobs[0]
	if job.AnswerID != "answer-id" {
		t.Fatalf("expected answer id answer-id, got %q", job.AnswerID)
	}
	if job.AudioPath != "storage/audio/interview-id/question-id.webm" {
		t.Fatalf("expected audio path, got %q", job.AudioPath)
	}
}

func TestUploadAnswerCompletesInterviewAfterSavingAnswer(t *testing.T) {
	repository := &stubAnswerRepository{
		answer: model.Answer{
			ID:          "answer-id",
			InterviewID: "interview-id",
			QuestionID:  "question-id",
			AudioPath:   stringPointer("storage/audio/interview-id/question-id.webm"),
		},
	}
	service := NewAnswerService(&stubAudioStorage{}, repository)

	_, err := service.UploadAnswer(context.Background(), UploadAnswerInput{
		InterviewID: "interview-id",
		QuestionID:  "question-id",
		ContentType: "audio/webm",
		Audio:       strings.NewReader("webm-bytes"),
	})

	if err != nil {
		t.Fatalf("UploadAnswer returned error: %v", err)
	}
	if repository.completeInterviewID != "interview-id" {
		t.Fatalf("expected completion check for interview-id, got %q", repository.completeInterviewID)
	}
}

func TestUploadAnswerRequiresAudio(t *testing.T) {
	service := NewAnswerService(&stubAudioStorage{}, &stubAnswerRepository{})

	_, err := service.UploadAnswer(context.Background(), UploadAnswerInput{
		InterviewID: "interview-id",
		QuestionID:  "question-id",
		ContentType: "audio/webm",
	})

	if !errors.Is(err, ErrAudioFileRequired) {
		t.Fatalf("expected ErrAudioFileRequired, got %v", err)
	}
}

func TestUploadAnswerRejectsUnsupportedAudioType(t *testing.T) {
	service := NewAnswerService(&stubAudioStorage{}, &stubAnswerRepository{})

	_, err := service.UploadAnswer(context.Background(), UploadAnswerInput{
		InterviewID: "interview-id",
		QuestionID:  "question-id",
		ContentType: "audio/wav",
		Audio:       strings.NewReader("wav-bytes"),
	})

	if !errors.Is(err, ErrUnsupportedAudioType) {
		t.Fatalf("expected ErrUnsupportedAudioType, got %v", err)
	}
}

func TestUploadAnswerWrapsStorageFailure(t *testing.T) {
	repository := &stubAnswerRepository{}
	service := NewAnswerService(&stubAudioStorage{err: errors.New("disk full")}, repository)

	_, err := service.UploadAnswer(context.Background(), UploadAnswerInput{
		InterviewID: "interview-id",
		QuestionID:  "question-id",
		ContentType: "audio/webm",
		Audio:       strings.NewReader("webm-bytes"),
	})

	if !errors.Is(err, ErrSaveAnswerAudioFailed) {
		t.Fatalf("expected ErrSaveAnswerAudioFailed, got %v", err)
	}
	if repository.completeInterviewID != "" {
		t.Fatalf("expected completion check to be skipped, got %q", repository.completeInterviewID)
	}
}

func TestUploadAnswerValidatesQuestionBeforeSavingAudio(t *testing.T) {
	storage := &stubAudioStorage{}
	service := NewAnswerService(storage, &stubAnswerRepository{validateErr: ErrQuestionNotFoundForInterview})

	_, err := service.UploadAnswer(context.Background(), UploadAnswerInput{
		InterviewID: "interview-id",
		QuestionID:  "question-id",
		ContentType: "audio/webm",
		Audio:       strings.NewReader("webm-bytes"),
	})

	if !errors.Is(err, ErrQuestionNotFoundForInterview) {
		t.Fatalf("expected ErrQuestionNotFoundForInterview, got %v", err)
	}
	if storage.interviewID != "" {
		t.Fatalf("expected storage not to be called, got interview id %q", storage.interviewID)
	}
}

func TestUploadAnswerPropagatesRepositoryFailure(t *testing.T) {
	expectedErr := errors.New("database unavailable")
	repository := &stubAnswerRepository{err: expectedErr}
	service := NewAnswerService(&stubAudioStorage{}, repository)

	_, err := service.UploadAnswer(context.Background(), UploadAnswerInput{
		InterviewID: "interview-id",
		QuestionID:  "question-id",
		ContentType: "audio/webm",
		Audio:       strings.NewReader("webm-bytes"),
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
	if repository.completeInterviewID != "" {
		t.Fatalf("expected completion check to be skipped, got %q", repository.completeInterviewID)
	}
}

type stubAudioStorage struct {
	interviewID string
	questionID  string
	audioBytes  string
	audioPath   string
	err         error
}

func (s *stubAudioStorage) SaveAnswerAudio(ctx context.Context, interviewID string, questionID string, file io.Reader) (string, error) {
	s.interviewID = interviewID
	s.questionID = questionID

	bytes, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	s.audioBytes = string(bytes)

	if s.err != nil {
		return "", s.err
	}
	if s.audioPath != "" {
		return s.audioPath, nil
	}
	return "storage/audio/" + interviewID + "/" + questionID + ".webm", nil
}

func (s *stubAudioStorage) DeleteAnswerAudio(ctx context.Context, audioPath string) error {
	return s.err
}

type stubAnswerRepository struct {
	audioPath           string
	answer              model.Answer
	validateErr         error
	err                 error
	completeInterviewID string
	completeErr         error
}

func (r *stubAnswerRepository) EnsureQuestionForInterview(ctx context.Context, interviewID string, questionID string) error {
	return r.validateErr
}

func (r *stubAnswerRepository) UpsertAnswer(ctx context.Context, interviewID string, questionID string, audioPath string) (model.Answer, error) {
	r.audioPath = audioPath
	if r.err != nil {
		return model.Answer{}, r.err
	}
	return r.answer, nil
}

func (r *stubAnswerRepository) CompleteInterviewIfAllQuestionsAnswered(ctx context.Context, interviewID string) error {
	r.completeInterviewID = interviewID
	return r.completeErr
}

func stringPointer(value string) *string {
	return &value
}

type stubAnswerAnalysisQueue struct {
	jobs []AnswerAnalysisJob
}

func (q *stubAnswerAnalysisQueue) Enqueue(job AnswerAnalysisJob) {
	q.jobs = append(q.jobs, job)
}
