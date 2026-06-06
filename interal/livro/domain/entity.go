package domain

import "time"

type Livro struct {
	ID        int64
	Titulo    string
	Autor     string
	IsActive  bool
	CreatedAt time.Time
}
