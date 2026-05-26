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
	questionGenerator := questionGeneratorForConfig(cfg)
	interviewService := service.NewInterviewService(questionGenerator, interviewRepository)
	interviewHandler := handler.NewInterviewHandler(interviewService)

	log.Println("starting interview-ai backend on :8080")
	if err := http.ListenAndServe(":8080", newRouter(interviewHandler)); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func questionGeneratorForConfig(cfg config.Config) llm.QuestionGenerator {
	if cfg.OpenAIAPIKey == "" {
		log.Println("OPENAI_API_KEY is not configured; using mock question generator")
		return llm.MockQuestionGenerator{}
	}

	log.Printf("OPENAI_API_KEY is configured; using OpenAI question generator with model %s", cfg.OpenAIModel)
	return llm.NewOpenAIQuestionGenerator(llm.OpenAIQuestionGeneratorConfig{
		APIKey: cfg.OpenAIAPIKey,
		Model:  cfg.OpenAIModel,
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
