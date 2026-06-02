package service

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"interview-ai/backend/internal/model"
)

func TestQuestionTTSServiceGeneratesSpeechForInterviewQuestion(t *testing.T) {
	repository := &stubQuestionTTSRepository{
		detail: model.InterviewDetail{
			ID:               "interview-id",
			QuestionLanguage: model.QuestionLanguageZhTW,
			Questions: []model.Question{
				{ID: "question-id", Text: "問題一", Order: 1},
			},
		},
	}
	generator := &stubQuestionTTSGenerator{audio: []byte("wav-bytes")}
	service := NewQuestionTTSService(repository, generator)

	audio, err := service.GenerateQuestionSpeech(context.Background(), "interview-id", "question-id")
	if err != nil {
		t.Fatalf("GenerateQuestionSpeech returned error: %v", err)
	}

	if !bytes.Equal(audio, []byte("wav-bytes")) {
		t.Fatalf("expected generated audio bytes, got %q", string(audio))
	}
	if generator.input.InterviewID != "interview-id" {
		t.Fatalf("expected interview id, got %q", generator.input.InterviewID)
	}
	if generator.input.QuestionID != "question-id" {
		t.Fatalf("expected question id, got %q", generator.input.QuestionID)
	}
	if generator.input.QuestionText != "問題一" {
		t.Fatalf("expected question text, got %q", generator.input.QuestionText)
	}
	if generator.input.QuestionLanguage != model.QuestionLanguageZhTW {
		t.Fatalf("expected zh-TW language, got %q", generator.input.QuestionLanguage)
	}
}

func TestQuestionTTSServiceRejectsQuestionOutsideInterview(t *testing.T) {
	repository := &stubQuestionTTSRepository{
		detail: model.InterviewDetail{
			ID: "interview-id",
			Questions: []model.Question{
				{ID: "other-question", Text: "問題二", Order: 2},
			},
		},
	}
	generator := &stubQuestionTTSGenerator{}
	service := NewQuestionTTSService(repository, generator)

	_, err := service.GenerateQuestionSpeech(context.Background(), "interview-id", "question-id")

	if !errors.Is(err, ErrQuestionNotFoundForInterview) {
		t.Fatalf("expected ErrQuestionNotFoundForInterview, got %v", err)
	}
	if generator.called {
		t.Fatal("expected generator not to be called for missing question")
	}
}

type stubQuestionTTSRepository struct {
	detail model.InterviewDetail
	err    error
}

func (r *stubQuestionTTSRepository) GetByID(ctx context.Context, interviewID string) (model.InterviewDetail, error) {
	if r.err != nil {
		return model.InterviewDetail{}, r.err
	}
	return r.detail, nil
}

type stubQuestionTTSGenerator struct {
	input             model.QuestionTTSInput
	audio             []byte
	audioByQuestionID map[string][]byte
	err               error
	called            bool
}

func (g *stubQuestionTTSGenerator) GenerateQuestionSpeech(ctx context.Context, input model.QuestionTTSInput) ([]byte, error) {
	g.called = true
	g.input = input
	if g.err != nil {
		return nil, g.err
	}
	if audio, ok := g.audioByQuestionID[input.QuestionID]; ok {
		return audio, nil
	}
	return g.audio, nil
}

func TestQuestionTTSServiceMapsRepositoryNoRowsToInterviewNotFound(t *testing.T) {
	repository := &stubQuestionTTSRepository{err: ErrInterviewNotFound}
	service := NewQuestionTTSService(repository, &stubQuestionTTSGenerator{})

	_, err := service.GenerateQuestionSpeech(context.Background(), "missing", "question-id")

	if !errors.Is(err, ErrInterviewNotFound) {
		t.Fatalf("expected ErrInterviewNotFound, got %v", err)
	}
}

func TestQuestionTTSServiceUsesDefaultLanguageWhenMissing(t *testing.T) {
	repository := &stubQuestionTTSRepository{
		detail: model.InterviewDetail{
			ID: "interview-id",
			Questions: []model.Question{
				{ID: "question-id", Text: "問題一", CreatedAt: time.Now()},
			},
		},
	}
	generator := &stubQuestionTTSGenerator{audio: []byte("wav-bytes")}
	service := NewQuestionTTSService(repository, generator)

	_, err := service.GenerateQuestionSpeech(context.Background(), "interview-id", "question-id")
	if err != nil {
		t.Fatalf("GenerateQuestionSpeech returned error: %v", err)
	}

	if generator.input.QuestionLanguage != model.QuestionLanguageZhTW {
		t.Fatalf("expected default zh-TW language, got %q", generator.input.QuestionLanguage)
	}
}

func TestQuestionTTSServiceGeneratesSpeechForAllInterviewQuestions(t *testing.T) {
	repository := &stubQuestionTTSRepository{
		detail: model.InterviewDetail{
			ID:               "interview-id",
			QuestionLanguage: model.QuestionLanguageZhTW,
			Questions: []model.Question{
				{ID: "question-1", Text: "問題一", Order: 1},
				{ID: "question-2", Text: "問題二", Order: 2},
			},
		},
	}
	generator := &stubQuestionTTSGenerator{audioByQuestionID: map[string][]byte{
		"question-1": []byte("question-1-wav"),
		"question-2": []byte("question-2-wav"),
	}}
	service := NewQuestionTTSService(repository, generator)

	audio, err := service.GenerateInterviewQuestionSpeech(context.Background(), "interview-id")
	if err != nil {
		t.Fatalf("GenerateInterviewQuestionSpeech returned error: %v", err)
	}

	if len(audio) != 2 {
		t.Fatalf("expected two audio results, got %+v", audio)
	}
	if audio[0].QuestionID != "question-1" || !bytes.Equal(audio[0].Audio, []byte("question-1-wav")) {
		t.Fatalf("unexpected first audio result: %+v", audio[0])
	}
	if audio[1].QuestionID != "question-2" || !bytes.Equal(audio[1].Audio, []byte("question-2-wav")) {
		t.Fatalf("unexpected second audio result: %+v", audio[1])
	}
}
