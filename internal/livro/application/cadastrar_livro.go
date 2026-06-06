package application

import (
	"context"
	"golang-variadric/internal/livro/domain"
)

func (u *livroUseCase) Cadastrar(ctx context.Context, titulo, author string, opts ...Option) (*domain.Livro, error) {
	if titulo == "" || author == "" {
		return nil, ErrInvalidInput
	}

	options := ResolveOptions(opts...)

	livro := &domain.Livro{
		Titulo:    titulo,
		Autor:     author,
		IsDigital: options.IsDigital,
		IsActive:  true,
	}
	if err := u.repository.Create(ctx, livro); err != nil {
		return nil, err
	}

	return livro, nil
}
