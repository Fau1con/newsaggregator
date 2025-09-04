package fetcher

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// HTTPFetcher реализует интерфейс FeedFetcher для загрузки RSS-лент по HTTP.
// Содержит HTTP-клиент для выполнения запросов и логгер для записи событий.
// Обеспечивает обработку ошибок сети, таймаутов и HTTP-статусов.
type HTTPFetcher struct {
	client *http.Client
	log    *slog.Logger
}

// NewHTTPFetcher создает новый экземпляр HTTPFetcher для загрузки RSS-лент.
// Использует стандартный HTTP-клиент и переданный логгер для записи событий.
func NewHTTPFetcher(log *slog.Logger) *HTTPFetcher {
	return &HTTPFetcher{
		client: http.DefaultClient,
		log:    log,
	}
}

// Fetch выполняет HTTP-запрос для получения RSS-ленты по указанному URL.
// Принимает контекст для контроля времени выполнения и отмены операции.
// Возвращает тело ответа как io.ReadCloser, которое должно быть закрыто после использования.
// В случае ошибки возвращает детальное описание проблемы с учетом HTTP-статуса и сетевых ошибок.
func (f *HTTPFetcher) Fetch(ctx context.Context, url string) (io.ReadCloser, error) {
	log := f.log.With(slog.String("url", url))
	log.Info("Fetching URL")
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Error("Failed to create HTTP request", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create request for url %s: %w", url, err)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		log.Error(
			"HTTP request failed",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to fetch url %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		log.Error(
			"Unexpected status code",
			slog.Int("status_code", resp.StatusCode),
		)
		return nil, fmt.Errorf("unexpected status code: %d for url %s", resp.StatusCode, url)
	}
	log.Info("Successfully fetched URL", slog.String("url", url))
	return resp.Body, nil
}
