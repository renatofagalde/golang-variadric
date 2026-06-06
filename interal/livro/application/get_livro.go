package application

import (
	"context"
	"golang-variadric/interal/livro/domain"
)

func (u *livroUseCase) GetByID(ctx context.Context, id int64) (*domain.Livro, error) {
	if id <= 0 {
		return nil, ErrInvalidInput
	}

	return u.repository.GetByID(ctx, id)
}
