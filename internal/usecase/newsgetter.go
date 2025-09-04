package usecase

import (
	"context"
	"news/internal/domain"
)

// NewsStorage определяет интерфейс для получения новостей из хранилища.
// Используется для предоставления данных через API.
type NewsStorage interface {
	GetNews(ctx context.Context, n int) ([]domain.Item, error)
}

// NewsGetterUseCase реализует бизнес-логику получения новостей для API.
// Предоставляет методы для доступа к сохраненным новостям.
type NewsGetterUseCase struct {
	storage NewsStorage
}

// NewNewsGetterUseCase создает новый экземпляр UseCase для получения новостей.
// Принимает реализацию хранилища для доступа к данным.
func NewNewsGetterUseCase(s NewsStorage) *NewsGetterUseCase {
	return &NewsGetterUseCase{storage: s}
}

// GetNews возвращает список новостей с ограничением по количеству.
// Делегирует вызов хранилищу и возвращает результат без дополнительной обработки.
func (us *NewsGetterUseCase) GetNews(ctx context.Context, limit int) ([]domain.Item, error) {
	return us.storage.GetNews(ctx, limit)
}
