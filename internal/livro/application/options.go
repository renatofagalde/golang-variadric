package application

import "golang-variadric/internal/livro/domain"

// CadastroOpts agrega as flags de configuração do cadastro.
// É o "saco de defaults": zero value = livro físico.
type CadastroOptions struct {
	IsDigital bool
	domain.FileType
}

// Option configura uma operação de cadastro.
// Implementada como functional option (idiomático em Go).
type Option func(options *CadastroOptions)

// Marcar o livro sendo Digital
func MarcarLivroSendoDigital() Option {
	return func(o *CadastroOptions) {
		o.IsDigital = true
	}
}

func MarcarTipo(fileTye domain.FileType) Option {
	return func(o *CadastroOptions) {
		o.IsDigital = true
		o.FileType = fileTye
	}
}

// ResolveOptions aplica todas as opções e retorna a config resultante.
// 1) começa com os DEFAULTS  2) aplica cada opção na ordem.
func ResolveOptions(opts ...Option) CadastroOptions {
	config := CadastroOptions{}
	for _, opt := range opts {
		opt(&config)
	}
	return config
}
