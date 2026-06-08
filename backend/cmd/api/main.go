package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"interview-ai/backend/internal/bootstrap"
	"interview-ai/backend/internal/config"
	"interview-ai/backend/internal/handler"
	"interview-ai/backend/internal/repository"
	"interview-ai/backend/internal/server"
	"interview-ai/backend/internal/service"
	"interview-ai/backend/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	configureLogger("info")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}
	configureLogger(cfg.LogLevel)

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	interviewRepository := repository.NewInterviewRepository(pool)
	answerRepository := repository.NewAnswerRepository(pool)
	llmCallLogRepository := repository.NewLLMCallLogRepository(pool)
	audioStorage := storage.NewLocalAudioStorage("storage/audio")
	questionGenerator := bootstrap.QuestionGeneratorForConfig(cfg, llmCallLogRepository)
	questionTTSGenerator := bootstrap.QuestionTTSGeneratorForConfig(cfg, llmCallLogRepository)
	answerAnalyzer := bootstrap.AnswerAnalyzerForConfig(cfg, llmCallLogRepository)
	answerAnalysisQueue := service.NewBackgroundAnswerAnalysisQueue(context.Background(), answerRepository, answerAnalyzer)
	answerAudioRetention := service.NewAnswerAudioRetentionService(answerRepository, audioStorage)
	service.StartAnswerAudioRetentionJob(context.Background(), answerAudioRetention, 72*time.Hour, time.Hour)
	interviewService := service.NewInterviewService(questionGenerator, interviewRepository, service.InterviewServiceConfig{
		CreationLimitPer24H: cfg.InterviewCreationLimitPer24H,
	})
	questionTTSService := service.NewQuestionTTSService(interviewRepository, questionTTSGenerator)
	answerService := service.NewAnswerService(audioStorage, answerRepository, answerAnalysisQueue)
	interviewHandler := handler.NewInterviewHandler(interviewService, answerService, questionTTSService, cfg.IPHashSalt)

	slog.Info("starting interview-ai backend", "addr", ":8080")
	if err := http.ListenAndServe(":8080", server.NewRouter(interviewHandler, "storage/audio")); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func configureLogger(level string) {
	levelVar := new(slog.LevelVar)
	switch level {
	case "debug":
		levelVar.Set(slog.LevelDebug)
	case "warn":
		levelVar.Set(slog.LevelWarn)
	case "error":
		levelVar.Set(slog.LevelError)
	default:
		levelVar.Set(slog.LevelInfo)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: levelVar,
	})))
}
