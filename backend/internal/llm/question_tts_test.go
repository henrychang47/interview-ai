package llm

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"net/http"
	"strings"
	"testing"

	"interview-ai/backend/internal/model"

	"google.golang.org/genai"
)

func TestGeminiQuestionTTSGeneratorReturnsWAVAudio(t *testing.T) {
	models := &fakeGeminiModels{
		results: []fakeGeminiResult{
			{audioBytes: []byte{1, 2, 3, 4}},
		},
	}
	replaceGeminiModelsFactory(t, func(ctx context.Context, apiKey string) (geminiContentGenerator, error) {
		return models, nil
	})

	generator := NewGeminiQuestionTTSGenerator(GeminiQuestionTTSGeneratorConfig{
		APIKey:        "test-key",
		Model:         "gemini-3.1-flash-tts-preview",
		FallbackModel: "gemini-2.5-flash-preview-tts",
		Voice:         "Kore",
		Backoff:       noBackoff,
	})

	audio, err := generator.GenerateQuestionSpeech(context.Background(), model.QuestionTTSInput{
		InterviewID:      "interview-id",
		QuestionID:       "question-id",
		QuestionText:     "請介紹你做過的 Go API 專案。",
		QuestionLanguage: model.QuestionLanguageZhTW,
	})
	if err != nil {
		t.Fatalf("GenerateQuestionSpeech returned error: %v", err)
	}

	call := models.calls[0]
	if call.model != "gemini-3.1-flash-tts-preview" {
		t.Fatalf("expected primary TTS model, got %q", call.model)
	}
	if call.config.ResponseModalities[0] != "AUDIO" {
		t.Fatalf("expected AUDIO response modality, got %+v", call.config.ResponseModalities)
	}
	voice := call.config.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName
	if voice != "Kore" {
		t.Fatalf("expected Kore voice, got %q", voice)
	}
	prompt := call.contents[0].Parts[0].Text
	if !strings.Contains(prompt, "請朗讀以下面試題目") {
		t.Fatalf("expected TTS prompt instruction, got %q", prompt)
	}
	if !strings.Contains(prompt, "請介紹你做過的 Go API 專案。") {
		t.Fatalf("expected question text in prompt, got %q", prompt)
	}

	if !bytes.Equal(audio[:4], []byte("RIFF")) {
		t.Fatalf("expected WAV RIFF header, got %q", audio[:4])
	}
	if !bytes.Equal(audio[8:12], []byte("WAVE")) {
		t.Fatalf("expected WAV format header, got %q", audio[8:12])
	}
	if binary.LittleEndian.Uint32(audio[24:28]) != 24000 {
		t.Fatalf("expected 24000 Hz sample rate, got %d", binary.LittleEndian.Uint32(audio[24:28]))
	}
	if !bytes.Equal(audio[44:], []byte{1, 2, 3, 4}) {
		t.Fatalf("expected PCM bytes after WAV header, got %+v", audio[44:])
	}
}

func TestGeminiQuestionTTSGeneratorRequiresAPIKey(t *testing.T) {
	createCount := 0
	replaceGeminiModelsFactory(t, func(ctx context.Context, apiKey string) (geminiContentGenerator, error) {
		createCount++
		return &fakeGeminiModels{}, nil
	})

	generator := NewGeminiQuestionTTSGenerator(GeminiQuestionTTSGeneratorConfig{
		APIKey: "",
		Model:  "gemini-3.1-flash-tts-preview",
	})

	_, err := generator.GenerateQuestionSpeech(context.Background(), model.QuestionTTSInput{
		QuestionText: "問題一",
	})

	if !errors.Is(err, ErrGeminiAPIKeyRequired) {
		t.Fatalf("expected ErrGeminiAPIKeyRequired, got %v", err)
	}
	if createCount != 0 {
		t.Fatalf("expected no GenAI client creation without API key, got %d", createCount)
	}
}

func TestGeminiQuestionTTSGeneratorFallsBackAfterTransientFailures(t *testing.T) {
	models := &fakeGeminiModels{
		results: []fakeGeminiResult{
			{err: genai.APIError{Code: http.StatusServiceUnavailable, Status: "UNAVAILABLE"}},
			{err: genai.APIError{Code: http.StatusServiceUnavailable, Status: "UNAVAILABLE"}},
			{err: genai.APIError{Code: http.StatusServiceUnavailable, Status: "UNAVAILABLE"}},
			{audioBytes: []byte{9, 8}},
		},
	}
	replaceGeminiModelsFactory(t, func(ctx context.Context, apiKey string) (geminiContentGenerator, error) {
		return models, nil
	})

	generator := NewGeminiQuestionTTSGenerator(GeminiQuestionTTSGeneratorConfig{
		APIKey:        "test-key",
		Model:         "gemini-3.1-flash-tts-preview",
		FallbackModel: "gemini-2.5-flash-preview-tts",
		Voice:         "Kore",
		Backoff:       noBackoff,
	})

	audio, err := generator.GenerateQuestionSpeech(context.Background(), model.QuestionTTSInput{
		QuestionText: "問題一",
	})
	if err != nil {
		t.Fatalf("GenerateQuestionSpeech returned error: %v", err)
	}

	if len(models.calls) != 4 {
		t.Fatalf("expected 3 primary attempts and 1 fallback attempt, got %d", len(models.calls))
	}
	if models.calls[3].model != "gemini-2.5-flash-preview-tts" {
		t.Fatalf("expected fallback TTS model, got %q", models.calls[3].model)
	}
	if !bytes.Equal(audio[44:], []byte{9, 8}) {
		t.Fatalf("expected fallback PCM bytes after WAV header, got %+v", audio[44:])
	}
}

func TestGeminiQuestionTTSGeneratorLogsCalls(t *testing.T) {
	logger := &stubLLMCallLogger{}
	models := &fakeGeminiModels{
		results: []fakeGeminiResult{
			{audioBytes: []byte{1, 2}, inputTokens: 5, outputTokens: 6, totalTokens: 11},
		},
	}
	replaceGeminiModelsFactory(t, func(ctx context.Context, apiKey string) (geminiContentGenerator, error) {
		return models, nil
	})

	generator := NewGeminiQuestionTTSGenerator(GeminiQuestionTTSGeneratorConfig{
		APIKey:        "test-key",
		Model:         "gemini-3.1-flash-tts-preview",
		FallbackModel: "gemini-2.5-flash-preview-tts",
		Voice:         "Kore",
		Backoff:       noBackoff,
		Logger:        logger,
	})

	_, err := generator.GenerateQuestionSpeech(context.Background(), model.QuestionTTSInput{
		InterviewID:  "interview-id",
		QuestionID:   "question-id",
		QuestionText: "問題一",
	})
	if err != nil {
		t.Fatalf("GenerateQuestionSpeech returned error: %v", err)
	}

	if len(logger.logs) != 1 {
		t.Fatalf("expected one LLM call log, got %d", len(logger.logs))
	}
	log := logger.logs[0]
	if log.Operation != model.LLMOperationGenerateQuestionTTS {
		t.Fatalf("expected generate_question_tts operation, got %q", log.Operation)
	}
	if log.Model != "gemini-3.1-flash-tts-preview" {
		t.Fatalf("expected TTS model, got %q", log.Model)
	}
	if log.InterviewID == nil || *log.InterviewID != "interview-id" {
		t.Fatalf("expected interview id, got %+v", log.InterviewID)
	}
	if log.QuestionID == nil || *log.QuestionID != "question-id" {
		t.Fatalf("expected question id, got %+v", log.QuestionID)
	}
	if log.Status != model.LLMCallStatusSuccess {
		t.Fatalf("expected success status, got %q", log.Status)
	}
	if log.InputTokens == nil || *log.InputTokens != 5 {
		t.Fatalf("expected 5 input tokens, got %+v", log.InputTokens)
	}
}
