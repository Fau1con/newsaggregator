package migrations

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Migration struct {
	ID    string
	UpSQL string
}

var allMigrations = []Migration{
	{
		ID: "020231120120000_create_news_table",
		UpSQL: `
		CREATE TABLE news(
		id serial PRIMARY KEY,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		pub_date TIMESTAMPTZ NOT NULL,
		link TEXT UNIQUE NOT NULL
		);`,
	},
}

// Apply применяет все необходимые миграции к базе данных.
func Apply(ctx context.Context, log *slog.Logger, pool *pgxpool.Pool) error {
	log = log.With(slog.String("component", "migrations"))
	log.Info("Starting database migrations check...")
	_, err := pool.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS schema_migrations (
	id TEXT PRIMARY KEY
	);
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}
	rows, err := pool.Query(ctx, "SELECT id FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}
	appliedMigrations := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan migration id: %w", err)
		}
		appliedMigrations[id] = true
	}
	rows.Close()
	sort.Slice(allMigrations, func(i, j int) bool {
		return allMigrations[i].ID < allMigrations[j].ID
	})
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	appliedCount := 0
	for _, m := range allMigrations {
		if !appliedMigrations[m.ID] {
			log.Info("Applying migration", slog.String("id", m.ID))
			if _, err := tx.Exec(ctx, m.UpSQL); err != nil {
				return fmt.Errorf("failed to apply migration %s: %w", m.ID, err)
			}
			if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (id) VALUES ($1)", m.ID); err != nil {
				return fmt.Errorf("failed to record migration %s: %w", m.ID, err)
			}
			appliedCount++
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migrations tansaction: %w", err)
	}
	if appliedCount > 0 {
		log.Info("Database migrations applied successfully", slog.Int("count", appliedCount))
	} else {
		log.Info("Database is up to date, no new migrations found.")
	}
	return nil
}
