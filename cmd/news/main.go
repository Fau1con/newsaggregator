package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"news/internal/adapter/fetcher"
	"news/internal/adapter/parser"
	"news/internal/config"
	"news/internal/logger"
	"news/internal/migrations"
	httpserver "news/internal/transport/http"
	"news/internal/usecase"
	"news/storage"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("FATAL: could not load config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("FATAL: invalid config: %v", err)
	}
	appLogger, err := logger.New(cfg.Logger)
	if err != nil {
		log.Fatalf("FATAL: could not setup logger: %v", err)
	}
	slog.SetDefault(appLogger)
	slog.Info("Starting News Aggregator",
		slog.String("component", "app"),
		slog.Int("feed_count", len(cfg.App.FeedURLs)),
		slog.String("processing_interval", cfg.App.ProcessingInterval),
	)
	dbPool, err := pgxpool.New(context.Background(), cfg.Database.DSN())
	if err != nil {
		slog.Error("Database connection failed",
			slog.String("component", "database"),
			slog.Any("error", err),
		)
		os.Exit(1)
	}
	defer dbPool.Close()
	if err := dbPool.Ping(context.Background()); err != nil {
		slog.Error("Database ping failed",
			slog.String("component", "database"),
			slog.Any("error", err),
		)
		os.Exit(1)
	}
	slog.Info("Database connection established", slog.String("component", "database"))
	if err := migrations.Apply(context.Background(), appLogger, dbPool); err != nil {
		slog.Error("Database migration failed",
			slog.String("component", "database"),
			slog.Any("error", err),
		)
		os.Exit(1)
	}
	feedNames := make(map[string]string)
	urls := make([]string, 0, len(cfg.App.FeedURLs))
	for _, feed := range cfg.App.FeedURLs {
		feedNames[feed.URL] = feed.Name
		urls = append(urls, feed.URL)
	}
	dbStorage := storage.NewPostgresNewsDB(dbPool, cfg.App, appLogger)
	httpFetcher := fetcher.NewHTTPFetcher(appLogger)
	xmlParser := parser.NewXMLParser(appLogger)
	feedProcessor := usecase.NewFeedProcessingUseCase(httpFetcher, xmlParser, dbStorage, appLogger, feedNames)
	newsGetter := usecase.NewNewsGetterUseCase(dbStorage)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	processingInterval, _ := time.ParseDuration(cfg.App.ProcessingInterval)
	wg.Add(1)
	go runWorker(ctx, &wg, feedProcessor, urls, processingInterval)
	port, err := getFreePort(8080)
	if err != nil {
		slog.Error("No free port available",
			slog.String("component", "server"),
			slog.Any("error", err),
		)
		os.Exit(1)
	}
	cfg.Server.Address = fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", cfg.Server.Address)
	if err != nil {
		slog.Error("Failed to create listener",
			slog.String("component", "server"),
			slog.Any("error", err),
		)
		os.Exit(1)
	}
	defer ln.Close()
	slog.Info("HTTP server ready",
		slog.String("component", "server"),
		slog.String("address", ln.Addr().String()),
	)

	handler := httpserver.NewHandler(appLogger, newsGetter)
	router := httpserver.NewServer(appLogger, handler)
	httpServer := &http.Server{
		Addr:    cfg.Server.Address,
		Handler: router,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("HTTP server starting",
			slog.String("component", "server"),
			slog.String("address", httpServer.Addr),
		)
		if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failed",
				slog.String("component", "server"),
				slog.Any("error", err),
			)
			cancel()
		} else {
			slog.Info("HTTP server stopped", slog.String("component", "server"))
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-stop:
		slog.Info("Shutdown signal received",
			slog.String("component", "app"),
			slog.String("signal", sig.String()),
		)
		time.Sleep(500 * time.Millisecond)
	case <-ctx.Done():
		slog.Warn("Context cancelled, initiating shutdown", slog.String("component", "app"))
	}
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown failed",
			slog.String("component", "server"),
			slog.Any("error", err),
		)
	} else {
		slog.Info("HTTP server stopped gracefully", slog.String("component", "server"))
	}

	wg.Wait()
	slog.Info("Application stopped", slog.String("component", "app"))
}
func runWorker(
	ctx context.Context,
	wg *sync.WaitGroup,
	processor *usecase.FeedProcessingUseCase,
	urls []string,
	interval time.Duration,
) {
	defer wg.Done()
	slog.Info("Feed processing worker started",
		slog.String("component", "worker"),
		slog.String("interval", interval.String()),
		slog.Int("feed_count", len(urls)),
	)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	processAllFeeds(ctx, processor, urls)
	for {
		select {
		case <-ticker.C:
			processAllFeeds(ctx, processor, urls)
		case <-ctx.Done():
			slog.Info("Worker stopping", slog.String("component", "worker"))
			return
		}
	}
}
func processAllFeeds(ctx context.Context, processor *usecase.FeedProcessingUseCase, urls []string) {
	start := time.Now()
	slog.Info("Feed processing cycle started",
		slog.String("component", "worker"),
		slog.Int("feeds_to_process", len(urls)),
	)
	var wg sync.WaitGroup
	successCount := 0
	errorCount := 0
	var mu sync.Mutex
	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			opCtx, opCancel := context.WithTimeout(ctx, 30*time.Second)
			defer opCancel()

			if err := processor.ProcessFeed(opCtx, u); err != nil {
				mu.Lock()
				errorCount++
				mu.Unlock()
				slog.Error("Feed processing failed",
					slog.String("component", "worker"),
					slog.String("url", u),
					slog.Any("error", err),
				)
			} else {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(url)
	}
	wg.Wait()
	duration := time.Since(start)
	slog.Info("Feed processing cycle completed",
		slog.String("component", "worker"),
		slog.Int("successful", successCount),
		slog.Int("errors", errorCount),
		slog.Int("total", len(urls)),
		slog.Duration("duration", duration),
	)
}
func getFreePort(startPort int) (int, error) {
	for port := startPort; port < startPort+100; port++ {
		address := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", address)
		if err == nil {
			listener.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free ports found in range %d-%d", startPort, startPort+100)
}
