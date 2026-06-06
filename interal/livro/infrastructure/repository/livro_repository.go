package repository

import (
	"golang-variadric/interal/livro/domain"
	"sync"
)

type livroDAO struct {
	mu     sync.Mutex
	nextID int64
	data   map[int64]domain.Livro
}

func NewLivroRepository() domain.LivroRepository {
	return &livroDAO{
		nextID: 1,
		data:   make(map[int64]domain.Livro),
	}
}
