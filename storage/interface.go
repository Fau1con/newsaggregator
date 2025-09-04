package storage

import (
	"context"
	"news/internal/domain"
)

// Storage определяет общий интерфейс для работы с хранилищем новостей.
// Объединяет методы для сохранения и получения новостей, а также закрытия соединения.
type Storage interface {
	SaveNews(ctx context.Context, feed *domain.Feed) (int, error)
	GetNews(ctx context.Context, n int) ([]domain.Item, error)
	Close()
}
