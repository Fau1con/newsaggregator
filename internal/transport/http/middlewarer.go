package http

import (
	"log/slog"
	"net/http"
	"time"
)

// loggingMiddleware логирует информацию о каждом запросе.
func loggingMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			entry := log.With(
				slog.String("component", "http"),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			)
			entry.Info("request started")
			start := time.Now()

			next.ServeHTTP(w, r)

			entry.Info("request completed",
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}
