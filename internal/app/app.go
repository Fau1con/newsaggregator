package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"news/internal/adapter/fetcher"
	"news/internal/adapter/parser"
	"news/internal/config"
	"news/internal/logger"
	"news/internal/migrations"
	server "news/internal/transport/http"
	"news/internal/usecase"
	"news/internal/worker"
	"news/storage"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// App представляет основное приложение News Aggregator.
// Координирует работу всех компонентов: HTTP-сервера, воркера обработки RSS,
// базы данных и системы логирования. Обеспечивает graceful startup и shutdown.
type App struct {
	config   *config.Config
	logger   *slog.Logger
	server   *http.Server
	worker   *worker.Worker
	dbPool   *pgxpool.Pool
	stopChan chan os.Signal
	wg       sync.WaitGroup
}

// New создает и инициализирует новый экземпляр приложения News Aggregator.
// Выполняет настройку логгера, подключение к базе данных, применение миграций,
// инициализацию всех зависимостей и компонентов системы.
// Возвращает ошибку в случае сбоя любой из инициализационных процедур.
func New(cfg *config.Config) (*App, error) {
	appLogger, err := logger.New(cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to setup logger: %w", err)
	}
	slog.SetDefault(appLogger)
	dbPool, err := pgxpool.New(context.Background(), cfg.Database.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	if err := dbPool.Ping(context.Background()); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("database ping failed: %w", err)
	}
	if err := migrations.Apply(context.Background(), appLogger, dbPool); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("migrations failed: %w", err)
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

	handler := server.NewHandler(appLogger, newsGetter)

	router := server.NewServer(appLogger, handler)

	processInterval, err := time.ParseDuration(cfg.App.ProcessingInterval)
	if err != nil {
		return nil, fmt.Errorf("bad init app: %w", err)
	}

	worker := worker.New(feedProcessor, urls, processInterval, appLogger)

	server := &http.Server{
		Addr:    cfg.Server.Address,
		Handler: router,
	}
	return &App{
		config:   cfg,
		logger:   appLogger,
		server:   server,
		worker:   worker,
		dbPool:   dbPool,
		stopChan: make(chan os.Signal, 1),
	}, nil
}

// Run запускает основное приложение News Aggregator.
// Инициализирует и запускает воркер для обработки RSS-лент, HTTP-сервер для API,
// и обрабатывает сигналы завершения работы. Метод блокируется до получения
// сигнала завершения. Возвращает ошибку в случае неудачи при запуске сервера.
func (a *App) Run() error {
	a.logger.Info("Starting News Aggregator",
		slog.String("component", "app"),
		slog.Int("feed_count", len(a.worker.GetURLs())),
		slog.String("processing_interval", a.worker.GetInterval().String()),
	)
	a.worker.Start()
	a.wg.Add(1)
	listener, err := net.Listen("tcp", a.server.Addr)
	if err != nil {
		return fmt.Errorf("failed to create listner: %w", err)
	}
	defer listener.Close()
	a.logger.Info("HTTP server ready",
		slog.String("component", "server"),
		slog.String("address", listener.Addr().String()),
	)
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := a.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			a.logger.Error("HTTP server failed", slog.Any("error", err))
		}
	}()
	signal.Notify(a.stopChan, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-a.stopChan:
		a.logger.Info("Shutdown signal received",
			slog.String("component", "app"),
			slog.String("signal", sig.String()),
		)
	}
	return a.Shutdown()
}

// Shutdown выполняет graceful shutdown приложения.
// Останавливает воркер обработки RSS, завершает HTTP-сервер, закрывает соединение с БД
// и ожидает завершения всех горутин. Использует таймаут 10 секунд для завершения
// HTTP-сервера. Возвращает ошибку в случае проблем при завершении работы сервера.
func (a *App) Shutdown() error {
	a.logger.Info("Starting graceful shutdown")
	if a.worker != nil {
		a.worker.Stop()
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.logger.Error("HTTP server shutdown failed", slog.Any("error", err))
	}
	if a.dbPool != nil {
		a.dbPool.Close()
	}
	a.wg.Wait()
	a.logger.Info("Application stopped grasefully")
	return nil
}
