package worker

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// FeedProcessor определяет интерфейс для обработки отдельных RSS-лент.
// Используется для внедрения зависимости в воркер.
type FeedProcessor interface {
	ProcessFeed(ctx context.Context, url string) error
}

// Worker реализует фонового воркера для периодической обработки RSS-лент.
// Управляет расписанием обработки, параллельным выполнением и мониторингом состояния.
type Worker struct {
	processor FeedProcessor
	urls      []string
	interval  time.Duration
	log       *slog.Logger
	ctx       context.Context
	cancel    context.CancelFunc
}

// New создает нового воркера для обработки RSS-лент.
// Принимает процессор, список URL, интервал обработки и логгер.
func New(processor FeedProcessor, urls []string, interval time.Duration, log *slog.Logger) *Worker {
	return &Worker{
		processor: processor,
		urls:      urls,
		interval:  interval,
		log:       log,
	}
}

// Start запускает воркер в отдельной горутине.
// Инициализирует контекст с возможностью отмены и начинает цикл обработки.
func (w *Worker) Start() {
	w.ctx, w.cancel = context.WithCancel(context.Background())
	go w.run()
}

// Stop останавливает воркер путем отмены контекста.
// Гарантирует graceful shutdown текущих операций.
func (w *Worker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

// run выполняет основной цикл работы воркера.
// Запускает периодическую обработку лент по расписанию и обрабатывает сигналы остановки.
func (w *Worker) run() {
	w.log.Info("Feed processing worker started",
		slog.String("component", "worker"),
		slog.String("interval", w.interval.String()),
		slog.Int("feed_count", len(w.urls)),
	)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	w.processAllFeeds()
	for {
		select {
		case <-ticker.C:
			w.processAllFeeds()
		case <-w.ctx.Done():
			w.log.Info("Worker stopping", slog.String("component", "worker"))
			return
		}
	}
}

// processAllFeeds обрабатывает все RSS-ленты параллельно.
// Измеряет общее время выполнения, считает успешные и неудачные обработки.
// Использует WaitGroup для синхронизации и atomic операции для подсчета.
func (w *Worker) processAllFeeds() {
	start := time.Now()
	w.log.Info("Feed processing cycle started",
		slog.String("component", "worker"),
		slog.Int("feed_to_process", len(w.urls)),
	)
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64
	for _, url := range w.urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			if w.ctx.Err() != nil {
				return
			}
			opCtx, opCancel := context.WithTimeout(w.ctx, 30*time.Second)
			defer opCancel()
			if w.processor == nil {
				w.log.Error("processor no init")
				return
			}
			if err := w.processor.ProcessFeed(opCtx, u); err != nil {
				atomic.AddInt64(&errorCount, 1)
				w.log.Error("Feed processing failed",
					slog.String("component", "worker"),
					slog.String("url", u),
					slog.Any("error", err),
				)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(url)
	}
	wg.Wait()
	duration := time.Since(start)
	w.log.Info("Feed processing cycle completed",
		slog.String("component", "worker"),
		slog.Int("successful", int(successCount)),
		slog.Int("errors", int(errorCount)),
		slog.Int("total", len(w.urls)),
		slog.Duration("duration", duration),
	)
}

// GetURLs возвращает список URL, которые обрабатывает воркер.
func (w *Worker) GetURLs() []string { return w.urls }

// GetInterval возвращает интервал обработки RSS-лент.
func (w *Worker) GetInterval() time.Duration { return w.interval }
