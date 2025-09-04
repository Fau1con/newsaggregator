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

// newsGetter определяет интерфейс для получения новостей из хранилища.
// Используется для внедрения зависимости и обеспечения тестируемости.
type newsGetter interface {
	GetNews(ctx context.Context, limit int) ([]domain.Item, error)
}

// Handler обрабатывает HTTP-запросы к API новостного агрегатора.
// Содержит логгер и зависимость для получения новостей из хранилища.
type Handler struct {
	log        *slog.Logger
	newsGetter newsGetter
}

// NewHandler создает новый экземпляр HTTP-обработчика.
// Принимает логгер для записи событий и реализацию интерфейса newsGetter.
func NewHandler(log *slog.Logger, getter newsGetter) *Handler {
	return &Handler{
		log:        log,
		newsGetter: getter,
	}
}

// getNews обрабатывает GET запросы к эндпоинту /api/news.
// Поддерживает параметр limit для ограничения количества возвращаемых новостей.
// Валидирует параметры запроса и возвращает новости в формате JSON.
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

// healthCheck обрабатывает запросы к эндпоинту /api/health.
// Возвращает статус работы сервиса в формате JSON.
// Используется для мониторинга и проверки доступности сервиса.
func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// respondWithError отправляет HTTP-ответ с ошибкой в формате JSON.
// Устанавливает соответствующий статус код и Content-Type.
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON отправляет HTTP-ответ с данными в формате JSON.
// Маршалит переданные данные в JSON и устанавливает заголовки.
// В случае ошибки маршалинга возвращает внутреннюю ошибку сервера.
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

// getRequestID генерирует уникальный идентификатор запроса на основе текущего времени.
// Используется для трассировки запросов в логах.
func getRequestID(ctx context.Context) string {
	return "req-" + time.Now().Format("20060102150405")
}
