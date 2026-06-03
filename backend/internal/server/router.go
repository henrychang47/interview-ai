package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func NewRouter(interviewHandler http.Handler, audioDir string) http.Handler {
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
