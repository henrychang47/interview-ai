package bootstrap

import (
	"context"
	"errors"
	"testing"

	"interview-ai/backend/internal/config"
	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"
	"interview-ai/backend/internal/service"
)

func TestQuestionGeneratorForConfigUsesMockWithoutGeminiKey(t *testing.T) {
	generator := QuestionGeneratorForConfig(config.Config{
		GeminiAPIKey:        "",
		GeminiModel:         "gemini-2.5-flash",
		GeminiFallbackModel: "gemini-2.5-flash-lite",
	}, nil)

	if _, ok := generator.(llm.MockQuestionGenerator); !ok {
		t.Fatalf("expected MockQuestionGenerator, got %T", generator)
	}
}

func TestQuestionGeneratorForConfigUsesGeminiWithGeminiKey(t *testing.T) {
	generator := QuestionGeneratorForConfig(config.Config{
		GeminiAPIKey:        "test-key",
		GeminiModel:         "gemini-2.5-flash",
		GeminiFallbackModel: "gemini-2.5-flash-lite",
	}, nil)

	if _, ok := generator.(*llm.GeminiQuestionGenerator); !ok {
		t.Fatalf("expected GeminiQuestionGenerator, got %T", generator)
	}
}

func TestAnswerAnalyzerForConfigUsesMockWithoutGeminiKey(t *testing.T) {
	analyzer := AnswerAnalyzerForConfig(config.Config{
		GeminiAPIKey:              "",
		GeminiAnswerModel:         "gemini-2.5-flash",
		GeminiAnswerFallbackModel: "gemini-2.5-flash-lite",
	}, nil)

	if _, ok := analyzer.(llm.MockAnswerAnalyzer); !ok {
		t.Fatalf("expected MockAnswerAnalyzer, got %T", analyzer)
	}
}

func TestAnswerAnalyzerForConfigUsesGeminiWithGeminiKey(t *testing.T) {
	analyzer := AnswerAnalyzerForConfig(config.Config{
		GeminiAPIKey:              "test-key",
		GeminiAnswerModel:         "gemini-2.5-flash",
		GeminiAnswerFallbackModel: "gemini-2.5-flash-lite",
	}, nil)

	if _, ok := analyzer.(*llm.GeminiAnswerAnalyzer); !ok {
		t.Fatalf("expected GeminiAnswerAnalyzer, got %T", analyzer)
	}
}

func TestQuestionTTSGeneratorForConfigUsesUnavailableWithoutGeminiKey(t *testing.T) {
	generator := QuestionTTSGeneratorForConfig(config.Config{
		GeminiAPIKey:           "",
		GeminiTTSModel:         "gemini-3.1-flash-tts-preview",
		GeminiTTSFallbackModel: "gemini-2.5-flash-preview-tts",
		GeminiTTSVoice:         "Kore",
	}, nil)

	_, err := generator.GenerateQuestionSpeech(context.Background(), model.QuestionTTSInput{})
	if !errors.Is(err, service.ErrQuestionTTSUnavailable) {
		t.Fatalf("expected ErrQuestionTTSUnavailable, got %v", err)
	}
}

func TestQuestionTTSGeneratorForConfigUsesGeminiWithGeminiKey(t *testing.T) {
	generator := QuestionTTSGeneratorForConfig(config.Config{
		GeminiAPIKey:           "test-key",
		GeminiTTSModel:         "gemini-3.1-flash-tts-preview",
		GeminiTTSFallbackModel: "gemini-2.5-flash-preview-tts",
		GeminiTTSVoice:         "Kore",
	}, nil)

	if _, ok := generator.(*llm.GeminiQuestionTTSGenerator); !ok {
		t.Fatalf("expected GeminiQuestionTTSGenerator, got %T", generator)
	}
}
