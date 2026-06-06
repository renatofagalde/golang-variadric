package domain

import "context"

type LivroRepository interface {
	Create(ctx context.Context, livro *Livro) error
	GetByID(ctx context.Context, id int64) (*Livro, error)
}
