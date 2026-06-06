package repository

import (
	"context"
	"golang-variadric/internal/livro/domain"
)

func (dao *livroDAO) GetByID(ctx context.Context, id int64) (*domain.Livro, error) {
	dao.mu.Lock()
	defer dao.mu.Unlock()
	livro, ok := dao.data[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &livro, nil
}
