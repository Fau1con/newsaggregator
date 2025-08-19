package usecase

import (
	"context"
	"news/internal/domain"
)

type NewsStorage interface {
	GetNews(ctx context.Context, n int) ([]domain.Item, error)
}
type NewsGetterUseCase struct {
	storage NewsStorage
}

func NewNewsGetterUseCase(s NewsStorage) *NewsGetterUseCase {
	return &NewsGetterUseCase{storage: s}
}
func (us *NewsGetterUseCase) GetNews(ctx context.Context, limit int) ([]domain.Item, error) {
	return us.storage.GetNews(ctx, limit)
}
