package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"interview-ai/backend/internal/config"
	"interview-ai/backend/internal/handler"
	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/repository"
	"interview-ai/backend/internal/service"
	"interview-ai/backend/internal/storage"

	"github.com/go-chi/chi/v5"
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
	questionGenerator := questionGeneratorForConfig(cfg, llmCallLogRepository)
	answerAnalyzer := answerAnalyzerForConfig(cfg, llmCallLogRepository)
	answerAnalysisQueue := service.NewBackgroundAnswerAnalysisQueue(context.Background(), answerRepository, answerAnalyzer)
	interviewService := service.NewInterviewService(questionGenerator, interviewRepository)
	answerService := service.NewAnswerService(audioStorage, answerRepository, answerAnalysisQueue)
	interviewHandler := handler.NewInterviewHandler(interviewService, answerService)

	slog.Info("starting interview-ai backend", "addr", ":8080")
	if err := http.ListenAndServe(":8080", newRouter(interviewHandler, "storage/audio")); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func questionGeneratorForConfig(cfg config.Config, logger llm.CallLogger) llm.QuestionGenerator {
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

func answerAnalyzerForConfig(cfg config.Config, logger llm.CallLogger) service.AnswerAnalyzer {
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

func newRouter(interviewHandler http.Handler, audioDir string) http.Handler {
	router := chi.NewRouter()
	router.Use(requestLoggingMiddleware)

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	if audioDir != "" {
		audioFileServer := http.StripPrefix("/audio/", http.FileServer(http.Dir(audioDir)))
		router.Get("/audio/*", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "audio/webm")
			audioFileServer.ServeHTTP(w, r)
		})
	}

	if interviewHandler != nil {
		router.Mount("/api/interviews", interviewHandler)
	}

	return router
}

func requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		start := time.Now()

		next.ServeHTTP(recorder, r)

		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("write json response", "error", err)
	}
}
