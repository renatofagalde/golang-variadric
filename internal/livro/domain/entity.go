package domain

import "time"

type Livro struct {
	ID        int64
	Titulo    string
	Autor     string
	IsDigital bool
	IsActive  bool
	CreatedAt time.Time
}
