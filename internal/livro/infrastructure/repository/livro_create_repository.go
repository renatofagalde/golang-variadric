package repository

import (
	"context"
	"golang-variadric/internal/livro/domain"
	"time"
)

func (dao *livroDAO) Create(ctx context.Context, livro *domain.Livro) error {
	if livro == nil {
		return ErrInvalidInput
	}

	dao.mu.Lock()
	defer dao.mu.Unlock()
	livro.ID = dao.nextID
	livro.CreatedAt = time.Now().UTC()
	dao.nextID++

	dao.data[livro.ID] = *livro
	return nil
}
