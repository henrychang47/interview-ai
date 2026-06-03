package bootstrap

import (
	"context"
	"log/slog"

	"interview-ai/backend/internal/config"
	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"
	"interview-ai/backend/internal/service"
)

func QuestionGeneratorForConfig(cfg config.Config, logger llm.CallLogger) llm.QuestionGenerator {
	if cfg.GeminiAPIKey == "" {
		slog.Info("using mock question generator", "reason", "GEMINI_API_KEY not configured")
		return llm.MockQuestionGenerator{}
	}

	slog.Info("using Gemini question generator", "model", cfg.GeminiModel, "fallback_model", cfg.GeminiFallbackModel)
	return llm.NewGeminiQuestionGenerator(llm.GeminiQuestionGeneratorConfig{
		APIKey:        cfg.GeminiAPIKey,
		Model:         cfg.GeminiModel,
		FallbackModel: cfg.GeminiFallbackModel,
		Logger:        logger,
	})
}

func AnswerAnalyzerForConfig(cfg config.Config, logger llm.CallLogger) service.AnswerAnalyzer {
	if cfg.GeminiAPIKey == "" {
		slog.Info("using mock answer analyzer", "reason", "GEMINI_API_KEY not configured")
		return llm.MockAnswerAnalyzer{}
	}

	slog.Info("using Gemini answer analyzer", "model", cfg.GeminiAnswerModel, "fallback_model", cfg.GeminiAnswerFallbackModel)
	return llm.NewGeminiAnswerAnalyzer(llm.GeminiAnswerAnalyzerConfig{
		APIKey:        cfg.GeminiAPIKey,
		Model:         cfg.GeminiAnswerModel,
		FallbackModel: cfg.GeminiAnswerFallbackModel,
		Logger:        logger,
	})
}

func QuestionTTSGeneratorForConfig(cfg config.Config, logger llm.CallLogger) service.QuestionTTSGenerator {
	if cfg.GeminiAPIKey == "" {
		slog.Info("using unavailable Gemini question TTS", "reason", "GEMINI_API_KEY not configured")
		return unavailableQuestionTTSGenerator{}
	}

	slog.Info("using Gemini question TTS", "model", cfg.GeminiTTSModel, "fallback_model", cfg.GeminiTTSFallbackModel, "voice", cfg.GeminiTTSVoice)
	return llm.NewGeminiQuestionTTSGenerator(llm.GeminiQuestionTTSGeneratorConfig{
		APIKey:        cfg.GeminiAPIKey,
		Model:         cfg.GeminiTTSModel,
		FallbackModel: cfg.GeminiTTSFallbackModel,
		Voice:         cfg.GeminiTTSVoice,
		Logger:        logger,
	})
}

type unavailableQuestionTTSGenerator struct{}

func (unavailableQuestionTTSGenerator) GenerateQuestionSpeech(ctx context.Context, input model.QuestionTTSInput) ([]byte, error) {
	return nil, service.ErrQuestionTTSUnavailable
}
