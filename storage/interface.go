package storage

import (
	"context"
	"news/internal/domain"
)

type Storage interface {
	SaveNews(ctx context.Context, feed *domain.Feed) (int, error)
	GetNews(ctx context.Context, n int) ([]domain.Item, error)
	Close()
}
