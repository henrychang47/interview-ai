package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

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
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	interviewRepository := repository.NewInterviewRepository(pool)
	answerRepository := repository.NewAnswerRepository(pool)
	audioStorage := storage.NewLocalAudioStorage("storage/audio")
	questionGenerator := questionGeneratorForConfig(cfg)
	interviewService := service.NewInterviewService(questionGenerator, interviewRepository)
	answerService := service.NewAnswerService(audioStorage, answerRepository)
	interviewHandler := handler.NewInterviewHandler(interviewService, answerService)

	log.Println("starting interview-ai backend on :8080")
	if err := http.ListenAndServe(":8080", newRouter(interviewHandler)); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func questionGeneratorForConfig(cfg config.Config) llm.QuestionGenerator {
	if cfg.GeminiAPIKey == "" {
		log.Println("GEMINI_API_KEY is not configured; using mock question generator")
		return llm.MockQuestionGenerator{}
	}

	log.Printf("GEMINI_API_KEY is configured; using Gemini question generator with model %s and fallback model %s", cfg.GeminiModel, cfg.GeminiFallbackModel)
	return llm.NewGeminiQuestionGenerator(llm.GeminiQuestionGeneratorConfig{
		APIKey:        cfg.GeminiAPIKey,
		Model:         cfg.GeminiModel,
		FallbackModel: cfg.GeminiFallbackModel,
	})
}

func newRouter(interviewHandler http.Handler) http.Handler {
	router := chi.NewRouter()

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	if interviewHandler != nil {
		router.Mount("/api/interviews", interviewHandler)
	}

	return router
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json response: %v", err)
	}
}
