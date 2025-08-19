package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"news/internal/domain"
	"strconv"
	"time"
)

type newsGetter interface {
	GetNews(ctx context.Context, limit int) ([]domain.Item, error)
}
type Handler struct {
	log        *slog.Logger
	newsGetter newsGetter
}

func NewHandler(log *slog.Logger, getter newsGetter) *Handler {
	return &Handler{
		log:        log,
		newsGetter: getter,
	}
}

// getNews - хендлер для эндпоинта GET /api/news
func (h *Handler) getNews(w http.ResponseWriter, r *http.Request) {
	const op = "transport.http/getNews"
	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", getRequestID(r.Context())),
	)
	if r.Method != http.MethodGet {
		log.Warn("method not allowed")
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			log.Warn("invalid limit parameter", slog.String("limit", limitStr))
			respondWithError(w, http.StatusBadRequest, "Invalid 'limit' parameter")
			return
		}
	}

	news, err := h.newsGetter.GetNews(r.Context(), limit)
	if err != nil {
		log.Error("Failed to get news", slog.Any("error", err))
		respondWithError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	respondWithJSON(w, http.StatusOK, news)
}

// healthCheck - хендлер для проверки состояния сервиса
func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Вспомогательные функции для ответов
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Failed to marshal JSON response"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
func getRequestID(ctx context.Context) string {
	return "req-" + time.Now().Format("20060102150405")
}
