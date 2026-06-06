package domain

import (
	"time"
)

type Livro struct {
	ID        int64
	Titulo    string
	Autor     string
	IsDigital bool
	FileType  FileType
	IsActive  bool
	CreatedAt time.Time
}
