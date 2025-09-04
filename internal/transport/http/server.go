package http

import (
	"log/slog"
	"net/http"
	"path/filepath"
)

// NewServer создает и настраивает HTTP-сервер с роутингом и middleware.
// Регистрирует эндпоинты для API, статических файлов.
// Добавляет middleware для логирования и CORS.
func NewServer(log *slog.Logger, h *Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/news", h.getNews)
	mux.HandleFunc("/api/health", h.healthCheck)
	staticDir := "web/static/"
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
			return
		}
		http.NotFound(w, r)
	})
	var handler http.Handler = mux
	handler = loggingMiddleware(log)(handler)
	handler = corsMiddleware()(handler)
	return handler
}

// corsMiddleware создает middleware для обработки CORS (Cross-Origin Resource Sharing).
// Разрешает запросы с любого origin и обрабатывает preflight OPTIONS запросы.
// Устанавливает необходимые заголовки для кросс-доменных запросов.
func corsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
			//w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
