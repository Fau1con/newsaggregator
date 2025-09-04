package storage

import (
	"context"
	"fmt"
	"log/slog"
	"news/internal/config"
	"news/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresNewsDB реализует хранение новостей в PostgreSQL.
// Использует connection pool для эффективного управления соединениями.
type PostgresNewsDB struct {
	pool             *pgxpool.Pool
	log              *slog.Logger
	defaultNewsLimit int
}

// NewPostgresNewsDB создает новый экземпляр хранилища PostgreSQL.
// Принимает пул соединений, конфигурацию приложения и логгер.
func NewPostgresNewsDB(pool *pgxpool.Pool, appCfg config.AppConfig, log *slog.Logger) *PostgresNewsDB {
	log.Info("Initializing Postgres news storage")
	return &PostgresNewsDB{
		pool:             pool,
		log:              log,
		defaultNewsLimit: appCfg.DefaultNewsLimit,
	}
}

// Close закрывает пул соединений с базой данных.
// Должен вызываться при завершении работы приложения.
func (db *PostgresNewsDB) Close() {
	db.log.Info("Closing database connection pool")
	db.pool.Close()
}

// SaveNews сохраняет новости из RSS-ленты в базу данных.
// Использует батчевую вставку для эффективности и обработку конфликтов по ссылкам.
// Возвращает количество сохраненных элементов и ошибку в случае неудачи.
func (db *PostgresNewsDB) SaveNews(ctx context.Context, feed *domain.Feed) (int, error) {
	if len(feed.Items) == 0 {
		return 0, nil
	}
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		db.log.Error(
			"Failed to begin transaction",
			slog.Any("error", err),
		)
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(context.Background()); rollbackErr != nil {
				db.log.Error("Failed to rollback transaction", slog.Any("error", rollbackErr))
			}
		}
	}()
	batch := &pgx.Batch{}
	query := `
	INSERT INTO news (title, content, pub_date, link)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (link) DO NOTHING;
	`
	for _, item := range feed.Items {
		batch.Queue(
			query,
			item.Title,
			item.Description,
			item.PubDate,
			item.Link,
		)
	}
	batchResult := tx.SendBatch(ctx, batch)
	if err := batchResult.Close(); err != nil {
		db.log.Error(
			"Failed to execute batch",
			slog.Any("error", err),
		)
		return 0, fmt.Errorf("failed to execute batch: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		db.log.Error("Failed to commit transacion", slog.Any("error", err))
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return len(feed.Items), nil
}

// GetNews возвращает список новостей из базы данных с ограничением по количеству.
// Сортирует новости по дате публикации (новые сначала).
// Использует значение по умолчанию если передан невалидный лимит.
func (db *PostgresNewsDB) GetNews(ctx context.Context, n int) ([]domain.Item, error) {
	limit := n
	if limit <= 0 {
		limit = db.defaultNewsLimit
	}
	log := db.log.With(slog.Int("limit", limit))
	const op = "storage.postgres.GetNews"
	log = log.With(slog.String("op", op))
	query := `
	SELECT id, title, content, pub_date, link
	FROM news
	ORDER BY pub_date DESC
	LIMIT $1;
	`
	rows, err := db.pool.Query(ctx, query, limit)
	if err != nil {
		log.Error("Database query failed", slog.Any("error", err))
		return nil, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}
	if rows != nil {
		defer rows.Close()
	}
	defer rows.Close()
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Item, error) {
		var item domain.Item
		var id int
		err := row.Scan(
			&id,
			&item.Title,
			&item.Description,
			&item.PubDate,
			&item.Link,
		)
		return item, err
	})
	if err != nil {
		log.Error("Failed to collect rows", slog.Any("error", err))
		return nil, fmt.Errorf("%s: failed to scan row: %w", op, err)
	}
	log.Info("Successfully retrieved news items", slog.Int("count", len(items)))
	return items, nil
}
