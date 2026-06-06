package application

import (
	"context"
	"golang-variadric/interal/livro/domain"
)

type LivroUsecase interface {
	Cadastrar(ctx context.Context, titulo, autor string) (*domain.Livro, error)
	GetByID(ctx context.Context, id int64) (*domain.Livro, error)
}

type livroUseCase struct {
	repository domain.LivroRepository
}

func NewLivroService(repository domain.LivroRepository) LivroUsecase {
	return &livroUseCase{repository: repository}
}
