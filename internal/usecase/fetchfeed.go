package usecase

import (
	"context"
	"io"
	"news/internal/domain"
)

// FeedFetcher определяет интерфейс для загрузки данных RSS-лент из внешних источников.
// Возвращает io.ReadCloser который должен быть закрыт после использования.
type FeedFetcher interface {
	Fetch(ctx context.Context, url string) (io.ReadCloser, error)
}

// FeedParser определяет интерфейс для парсинга RSS-данных в доменную модель.
// Преобразует сырые данные в структурированные объекты Feed.
type FeedParser interface {
	Parse(ctx context.Context, reader io.Reader) (*domain.Feed, error)
}

// FeedStorage определяет интерфейс для сохранения новостей в постоянное хранилище.
// Возвращает количество сохраненных элементов и ошибку в случае неудачи.
type FeedStorage interface {
	SaveNews(ctx context.Context, feed *domain.Feed) (int, error)
}
