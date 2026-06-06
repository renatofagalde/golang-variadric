# Functional Options em Go
## Como adicionar parâmetro opcional sem quebrar quem já usa

Roteiro da apresentação. Três partes, cada uma na sua branch:

| Parte | Branch | Assunto |
|---|---|---|
| 1 | feature/regular-book-code | A versão funcional, assinatura fixa |
| 2 | feature/digital-book-support-and-types | Livro digital com functional options + enum de formato |
| 3 | feature/race-conditions-demo | Goroutines e WaitGroup provando o mutex |

O problema que vamos resolver: em .NET, parâmetro opcional é trivial
(`bool digital = false`). Go não tem default value nem overload de função.
Então como evoluir uma função que já está em produção sem quebrar
todo mundo que chama ela?

---

# PARTE 1: a versão em produção (feature/regular-book-code)

## 1.1 Projeto e estrutura de pastas

```bash
mkdir golang-variadric && cd golang-variadric
go mod init golang-variadric
git init -b main

mkdir -p cmd/cli
mkdir -p internal/livro/domain
mkdir -p internal/livro/application
mkdir -p internal/livro/infrastructure/repository
```

O que vamos construir:

```
golang-variadric/
├── go.mod
├── cmd/cli/main.go
└── internal/livro/
    ├── domain/
    │   ├── entity.go
    │   └── repository.go
    ├── application/
    │   ├── livro_usecase.go
    │   ├── errors.go
    │   ├── cadastrar_livro.go
    │   └── get_livro.go
    └── infrastructure/repository/
        ├── livro_repository.go
        ├── errors.go
        ├── livro_create_repository.go
        └── livro_get_by_id_repository.go
```

Por que assim: /cmd guarda os pontos de entrada, /internal é código
privado que nenhum módulo externo consegue importar (o compilador garante).
Domain não conhece application nem infrastructure. As dependências sempre
apontam para dentro.

## 1.2 Domain: a entidade

**internal/livro/domain/entity.go**

Entidade pura. Sem tag de GORM, sem tag de JSON. Quem precisa de tag
é model e DTO, nunca o domínio.

```go
package domain

import "time"

type Livro struct {
	ID        int64
	Titulo    string
	Autor     string
	IsActive  bool
	CreatedAt time.Time
}
```

## 1.3 Domain: o contrato do repositório

**internal/livro/domain/repository.go**

Interface pequena, dois métodos. Quem define a interface é quem consome,
não quem implementa.

```go
package domain

import "context"

type LivroRepository interface {
	Create(ctx context.Context, livro *Livro) error
	GetByID(ctx context.Context, id int64) (*Livro, error)
}
```

## 1.4 Repository: a struct e o construtor

**internal/livro/infrastructure/repository/livro_repository.go**

Em memória para a demo. No projeto real seria GORM com PostgreSQL.
Este arquivo tem só a struct e o construtor. Cada método vai ter
seu próprio arquivo.

```go
package repository

import (
	"golang-variadric/internal/livro/domain"
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
```

Sobre o mutex: map em Go não aguenta escrita concorrente (o programa
morre com `fatal error: concurrent map writes`), e o nextID++ é um
read-modify-write que perde incremento quando duas goroutines executam
ao mesmo tempo. Na Parte 3 a gente prova isso na prática. Por enquanto
ele fica aqui, escrito e correto.

## 1.5 Repository: os erros do pacote

**internal/livro/infrastructure/repository/errors.go**

Convenção: os sentinel errors do pacote ficam agrupados num errors.go.
Quem abre o pacote enxerga de cara quais erros ele pode devolver.
Texto em minúscula e sem ponto final, porque erro em Go aparece no meio
de mensagens encadeadas (o go vet inclusive reclama de maiúscula).

```go
package repository

import "errors"

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("livro not found")
)
```

## 1.6 Repository: método Create

**internal/livro/infrastructure/repository/livro_create_repository.go**

```go
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
```

## 1.7 Repository: método GetByID

**internal/livro/infrastructure/repository/livro_get_by_id_repository.go**

```go
package repository

import (
	"context"
	"golang-variadric/internal/livro/domain"
)

func (dao *livroDAO) GetByID(ctx context.Context, id int64) (*domain.Livro, error) {
	dao.mu.Lock()
	defer dao.mu.Unlock()
	livro, ok := dao.data[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &livro, nil
}
```

Detalhe do `return &livro`: ler do map devolve uma cópia da struct,
e o & pega o endereço dessa cópia. Em C isso seria ponteiro pendurado;
em Go o escape analysis percebe que o endereço escapa da função e aloca
na heap. E não dá para escrever `&dao.data[id]` direto, elemento de map
não é endereçável porque o map reorganiza as entradas internamente.

## 1.8 Application: interface e implementação

**internal/livro/application/livro_usecase.go**

```go
package application

import (
	"context"
	"golang-variadric/internal/livro/domain"
)

type LivroUsecase interface {
	Cadastrar(ctx context.Context, titulo, author string) (*domain.Livro, error)
	GetByID(ctx context.Context, id int64) (*domain.Livro, error)
}

type livroUseCase struct {
	repository domain.LivroRepository
}

func NewLivroService(repository domain.LivroRepository) LivroUsecase {
	return &livroUseCase{repository: repository}
}
```

Guardem essa assinatura: Cadastrar(ctx, titulo, author). É ela que
o time inteiro vai chamar e é ela que não pode quebrar depois.

## 1.9 Application: os erros do pacote

**internal/livro/application/errors.go**

```go
package application

import "errors"

var ErrInvalidInput = errors.New("invalid input")
```

## 1.10 Application: caso de uso Cadastrar

**internal/livro/application/cadastrar_livro.go**

Um caso de uso por arquivo.

```go
package application

import (
	"context"
	"golang-variadric/internal/livro/domain"
)

func (u *livroUseCase) Cadastrar(ctx context.Context, titulo, author string) (*domain.Livro, error) {
	if titulo == "" || author == "" {
		return nil, ErrInvalidInput
	}

	livro := &domain.Livro{
		Titulo:   titulo,
		Autor:    author,
		IsActive: true,
	}
	if err := u.repository.Create(ctx, livro); err != nil {
		return nil, err
	}

	return livro, nil
}
```

## 1.11 Application: caso de uso GetByID

**internal/livro/application/get_livro.go**

```go
package application

import (
	"context"
	"golang-variadric/internal/livro/domain"
)

func (u *livroUseCase) GetByID(ctx context.Context, id int64) (*domain.Livro, error) {
	if id <= 0 {
		return nil, ErrInvalidInput
	}

	return u.repository.GetByID(ctx, id)
}
```

## 1.12 Amarrando tudo no CLI

**cmd/cli/main.go**

```go
package main

import (
	"context"
	"fmt"
	"golang-variadric/internal/livro/application"
	"golang-variadric/internal/livro/infrastructure/repository"
	"log"
)

func main() {

	ctx := context.Background()

	livroRepository := repository.NewLivroRepository()
	service := application.NewLivroService(livroRepository)

	//codigo em producao:
	livroHobbit, err := service.Cadastrar(ctx, "O Hobbit", "Tolkien")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("created: %+v\n", *livroHobbit)

	livroCC, err := service.Cadastrar(ctx, "Clean Code", "Robert Martin")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("created: %+v\n", *livroCC)
}
```

## 1.13 Validar, commitar, versionar

```bash
go build -o bootstrap ./cmd/cli
./bootstrap
```

Saída esperada:

```
created: {ID:1 Titulo:O Hobbit Autor:Tolkien IsActive:true CreatedAt:...}
created: {ID:2 Titulo:Clean Code Autor:Robert Martin IsActive:true CreatedAt:...}
```

```bash
echo "bootstrap" >> .gitignore
git add .
git commit -m "✨ feat: bookstore v1 with fixed signature Cadastrar(titulo, author)"
git tag -a v1.0.0 -m "Functional version with fixed signature"
git checkout -b feature/regular-book-code
git checkout main
```

Pronto. Essa é a versão "em produção". A branch feature/regular-book-code
guarda o antes da comparação.

---

# PARTE 2: livro digital (feature/digital-book-support-and-types)

Chega o requisito: o livro agora pode ser digital, e quando for digital
tem um formato de arquivo (PDF, EPUB, TXT).

## 2.1 Por que não dá para só adicionar um parâmetro

A primeira ideia de quem vem de .NET:

```go
// quebra todo mundo
Cadastrar(ctx context.Context, titulo, author string, digital bool) (*domain.Livro, error)
```

Resultado: todas as chamadas existentes param de compilar.

```
not enough arguments in call to service.Cadastrar
	have (context.Context, string, string)
	want (context.Context, string, string, bool)
```

Mudar assinatura é breaking change. No semver isso obriga um v2.0.0
e todo consumidor precisa migrar. A saída idiomática do Go usa duas
peças: parâmetro variádico e functional options.

A base de tudo é que o variádico aceita zero argumentos:

```go
func soma(nums ...int) int { ... }

soma()           // válido, nums vira um slice vazio
soma(10, 20)     // válido
```

Como dá para chamar com zero argumentos, quem já chamava sem nada
continua compilando igual. É essa propriedade que vamos explorar.

## 2.2 Abrir a branch

```bash
git checkout -b feature/digital-book-support-and-types
```

## 2.3 Domain: o enum de formato

**internal/livro/domain/file_type.go** (arquivo novo)

Go não tem palavra-chave enum. O equivalente é tipo nomeado mais
constantes tipadas. Aqui usamos string por baixo, porque imprime e
serializa legível (sai "PDF" no log, não um número mágico).

Atenção ao lugar: o FileType fica no domain, não na application.
A entidade Livro vai ter um campo desse tipo, e se o tipo morasse na
application o entity.go precisaria importar application, que importa
domain, e o compilador barra com import cycle not allowed. Regra
prática: o tipo mora na camada mais interna que o usa.

```go
package domain

type FileType string

const (
	FileTypeNone FileType = "" //arquivo físico
	FileTypePDF  FileType = "PDF"
	FileTypeEPUB FileType = "EPUB"
	FileTypeTXT  FileType = "TXT"
)

func (f FileType) IsValid() bool {
	switch f {
	case FileTypeNone, FileTypePDF, FileTypeEPUB, FileTypeTXT:
		return true
	}
	return false
}
```

Repare que FileTypeNone é a string vazia, que é o zero value. Isso vai
encaixar de graça com os defaults do padrão. E o IsValid existe porque
enum em Go não é fechado: nada impede alguém de criar FileType("XYZ")
na marra, então o tipo sabe se validar. Receiver por valor, sem ponteiro,
porque o método só lê.

## 2.4 Domain: a entidade ganha os campos

**internal/livro/domain/entity.go**

```go
package domain

import "time"

type Livro struct {
	ID        int64
	Titulo    string
	Autor     string
	IsDigital bool
	FileType  FileType
	IsActive  bool
	CreatedAt time.Time
}
```

Zero value trabalhando a nosso favor: IsDigital false e FileType vazio
significam livro físico, que é exatamente o comportamento antigo.

## 2.5 Application: as options

**internal/livro/application/options.go** (arquivo novo)

A peça central do padrão. Em vez de receber um bool na assinatura,
o Cadastrar vai receber funções de configuração.

```go
package application

import "golang-variadric/internal/livro/domain"

// CadastroOptions agrega as configurações do cadastro.
// O zero value da struct são os defaults: livro físico.
type CadastroOptions struct {
	IsDigital bool
	FileType  domain.FileType
}

// Option é uma função que modifica as configurações.
type Option func(options *CadastroOptions)

// MarcarLivroSendoDigital marca o livro como digital.
func MarcarLivroSendoDigital() Option {
	return func(o *CadastroOptions) {
		o.IsDigital = true
	}
}

// MarcarTipo define o formato do arquivo. Ter formato implica ser digital.
func MarcarTipo(fileType domain.FileType) Option {
	return func(o *CadastroOptions) {
		o.IsDigital = true
		o.FileType = fileType
	}
}

// ResolveOptions parte dos defaults e aplica cada opção na ordem.
func ResolveOptions(opts ...Option) CadastroOptions {
	config := CadastroOptions{}
	for _, opt := range opts {
		opt(&config)
	}
	return config
}
```

Como explicar cada peça:

O type Option func(*CadastroOptions) é um tipo de função nomeado,
o delegate do .NET (Action de CadastroOptions). Uma Option é qualquer
função que recebe o ponteiro da config e mexe nela. Ponteiro porque
sem ele a função receberia uma cópia e a mudança se perderia.

MarcarLivroSendoDigital e MarcarTipo são fábricas: cada uma devolve
uma dessas funções. No MarcarTipo a closure captura o fileType que
o chamador passou.

ResolveOptions é o coração e tem três tempos: cria a struct com zero
values (os defaults, o equivalente do bool digital = false), roda
o loop executando cada opção em cima do ponteiro, devolve a config
pronta. Se ninguém passou opção, o loop nem roda e sobram os defaults.
As opções são aplicadas na ordem; se duas mexem no mesmo campo,
a última vence.

O nome desse padrão é Functional Options. Não está no catálogo do GoF;
é um idiomatismo do Go (Rob Pike e Dave Cheney escreveram os textos
de referência). É primo do Builder: resolve o mesmo problema de
construção com parâmetros opcionais, mas troca o objeto builder com
métodos fluentes por funções passadas como argumento.

Usamos exatamente esse padrão no module-header2object: o ValidationOpts
é o CadastroOptions, o SkipSite() é o MarcarLivroSendoDigital, e o
ResolveOptions é o mesmo. Catorze handlers continuaram chamando
IsValid() sem mudar uma linha quando a opção entrou.

## 2.6 Application: a interface ganha o variádico

**internal/livro/application/livro_usecase.go**

```go
package application

import (
	"context"
	"golang-variadric/internal/livro/domain"
)

type LivroUsecase interface {
	Cadastrar(ctx context.Context, titulo, author string, opts ...Option) (*domain.Livro, error)
	GetByID(ctx context.Context, id int64) (*domain.Livro, error)
}

type livroUseCase struct {
	repository domain.LivroRepository
}

func NewLivroService(repository domain.LivroRepository) LivroUsecase {
	return &livroUseCase{repository: repository}
}
```

Sobre a posição dos três pontos, que confunde todo mundo no começo:
na declaração eles vêm antes do tipo (opts ...Option, "aceito zero
ou mais"); na chamada eles vêm depois da variável (ResolveOptions(opts...),
"espalha esse slice como argumentos"). Declaração e spread, dois
contextos, duas posições. É o params do C# e o spread do array.

## 2.7 Application: o Cadastrar aplica as opções

**internal/livro/application/cadastrar_livro.go**

```go
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

	if !options.FileType.IsValid() {
		return nil, ErrInvalidInput
	}

	livro := &domain.Livro{
		Titulo:    titulo,
		Autor:     author,
		IsDigital: options.IsDigital,
		FileType:  options.FileType,
		IsActive:  true,
	}
	if err := u.repository.Create(ctx, livro); err != nil {
		return nil, err
	}

	return livro, nil
}
```

Sempre os mesmos três tempos: defaults, aplicar opções, construir.
A validação do formato fica aqui porque regra de negócio é
responsabilidade do caso de uso. E note o que não mudou: errors.go,
get_livro.go e o repositório inteiro ficaram intocados.

## 2.8 A prova: os dois mundos no mesmo main

**cmd/cli/main.go**

As duas chamadas antigas ficam exatamente como estavam. Só adicionamos
as novas no final.

```go
package main

import (
	"context"
	"fmt"
	"golang-variadric/internal/livro/application"
	"golang-variadric/internal/livro/domain"
	"golang-variadric/internal/livro/infrastructure/repository"
	"log"
)

func main() {

	ctx := context.Background()

	livroRepository := repository.NewLivroRepository()
	service := application.NewLivroService(livroRepository)

	//codigo em producao, nao mudou uma linha:
	livroHobbit, err := service.Cadastrar(ctx, "O Hobbit", "Tolkien")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("created: %+v\n", *livroHobbit)

	livroCC, err := service.Cadastrar(ctx, "Clean Code", "Robert Martin")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("created: %+v\n", *livroCC)

	//codigo novo, quem precisa do recurso opta por ele:
	livroEBook, err := service.Cadastrar(ctx, "Go in Action", "Ketelsen",
		application.MarcarLivroSendoDigital())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("created: %+v\n", *livroEBook)

	livroEBookPDF, err := service.Cadastrar(ctx, "Effective Go", "Go Team",
		application.MarcarTipo(domain.FileTypePDF))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("created: %+v\n", *livroEBookPDF)
}
```

## 2.9 Validar, commitar, versionar

```bash
go build -o bootstrap ./cmd/cli
./bootstrap
```

Saída esperada:

```
created: {ID:1 Titulo:O Hobbit Autor:Tolkien IsDigital:false FileType: IsActive:true CreatedAt:...}
created: {ID:2 Titulo:Clean Code Autor:Robert Martin IsDigital:false FileType: IsActive:true CreatedAt:...}
created: {ID:3 Titulo:Go in Action Autor:Ketelsen IsDigital:true FileType: IsActive:true CreatedAt:...}
created: {ID:4 Titulo:Effective Go Autor:Go Team IsDigital:true FileType:PDF IsActive:true CreatedAt:...}
```

Livros 1 e 2 são as chamadas antigas, saíram com os defaults.
Livro 3 optou pelo digital. Livro 4 optou pelo formato, que implica
digital. Nenhuma chamada antiga foi tocada.

```bash
git add .
git commit -m "✨ feat: add digital book support via functional options (backward compatible)"
git tag -a v1.1.0 -m "Add digital book support and file types (backward compatible)"
```

O semver conta a história sozinho: v1.0.0 para v1.1.0, bump minor,
porque é feature nova sem breaking change. O bool na assinatura teria
forçado v2.0.0. Mesmo motivo pelo qual o module-header2object foi de
v1.1 para v1.2 quando ganhou o SkipSite, e não para v2.

Para fechar, o diff entre as branches mostra o tamanho real da mudança:

```bash
git diff feature/regular-book-code feature/digital-book-support-and-types --stat
```

Esperado: entity.go, file_type.go (novo), options.go (novo),
livro_usecase.go, cadastrar_livro.go, main.go. Repositório, errors
e get_livro fora do diff.

Uma honestidade técnica para deixar registrada: para quem chama, a
mudança é cem por cento compatível. Mas como mexemos num método de
interface, mocks gerados com mockgen precisam ser regenerados. Em
método de struct, como o IsValid do header2object, nem isso acontece.

---

# PARTE 3: provando o mutex (feature/race-conditions-demo)

Lá na Parte 1 o repositório nasceu com mutex e ficou a promessa de
provar que ele é necessário. Agora: 50 goroutines cadastrando ao
mesmo tempo. O código do repositório não muda nada nesta parte,
só ganhamos um segundo ponto de entrada.

## 3.1 Abrir a branch

```bash
git checkout -b feature/race-conditions-demo
```

## 3.2 Segundo entrypoint

Cada subpasta de /cmd é um binário independente, então o main da
história principal fica quieto.

```bash
mkdir -p cmd/racedemo
```

**cmd/racedemo/main.go**

```go
package main

import (
	"context"
	"fmt"
	"golang-variadric/internal/livro/application"
	"golang-variadric/internal/livro/infrastructure/repository"
	"log"
	"sync"
)

func main() {

	ctx := context.Background()

	livroRepository := repository.NewLivroRepository()
	service := application.NewLivroService(livroRepository)

	const total = 50

	var wg sync.WaitGroup
	ids := make(chan int64, total)

	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			livro, err := service.Cadastrar(ctx,
				fmt.Sprintf("Livro %d", n), "Autor Concorrente")
			if err != nil {
				log.Println("error:", err)
				return
			}
			ids <- livro.ID
		}(i)
	}

	wg.Wait()
	close(ids)

	// com mutex, os 50 IDs saem únicos: nenhum nextID++ se perdeu
	seen := make(map[int64]bool)
	duplicated := 0
	for id := range ids {
		if seen[id] {
			duplicated++
		}
		seen[id] = true
	}

	fmt.Printf("created: %d | unique IDs: %d | duplicated: %d\n",
		total, len(seen), duplicated)
}
```

O que vale narrar enquanto digita:

O WaitGroup dá ciclo de vida claro às goroutines: Add antes de criar,
defer wg.Done() dentro, Wait() segura o main até as 50 terminarem.

O channel tem buffer do tamanho total para nenhuma goroutine bloquear
no envio. Channel também é uma forma segura de comunicar entre
goroutines, sem lock adicional aqui.

O close(ids) é o sinal de fim de stream. O for range em cima de channel
só sai do loop quando o channel está fechado e vazio. Sem o close,
o range esperaria o item 51 para sempre e o runtime mataria o programa
com deadlock detectado. E quem fecha é sempre o lado produtor, depois
do Wait, quando temos certeza de que ninguém mais vai enviar.

## 3.3 Rodar com o race detector

```bash
go run -race ./cmd/racedemo
```

Saída esperada, limpa:

```
created: 50 | unique IDs: 50 | duplicated: 0
```

## 3.4 A parte divertida (opcional, mudança temporária)

Comente as duas linhas de lock no livro_create_repository.go,
direto no editor:

```go
	// dao.mu.Lock()
	// defer dao.mu.Unlock()
```

E rode de novo:

```bash
go run -race ./cmd/racedemo
```

O detector acusa na hora:

```
==================
WARNING: DATA RACE
Write at 0x00c000... by goroutine 8:
  golang-variadric/internal/livro/infrastructure/repository.(*livroDAO).Create()
...
```

Dependendo do timing, o programa nem chega no warning: morre direto
com `fatal error: concurrent map writes`, que é o próprio map se
defendendo.

Restaure os locks antes de commitar, o código final mantém o mutex:

```bash
git checkout -- internal/
go run -race ./cmd/racedemo
```

## 3.5 Commitar

```bash
git add cmd/racedemo/
git commit -m "✨ feat: add race demo proving repository mutex under concurrency"
```

Dois ganchos para fechar a conversa com o time:

Rodar go test -race ./... deveria ser hábito, não evento.

No Lambda o paralelismo vem de instâncias concorrentes, não de
goroutines dentro do mesmo processo. Por isso lá a proteção
equivalente é lock no banco (SELECT FOR UPDATE em débito com
validação de saldo), não mutex em memória. O princípio é o mesmo,
read-modify-write sob concorrência precisa de exclusão; muda a
camada onde a exclusão acontece.

---

# Fechamento

```
feature/regular-book-code               v1.0.0  assinatura fixa, o antes
feature/digital-book-support-and-types  v1.1.0  functional options, o depois
feature/race-conditions-demo                    mutex à prova de 50 goroutines
```

O resumo em quatro frases:

Variádico aceita zero argumentos, e é isso que protege as chamadas antigas.

ResolveOptions faz o papel do default value do .NET: o zero value da
struct são os defaults, as opções só sobrescrevem o que o chamador pedir.

A assinatura nunca mais muda: formato novo, idioma, qualquer configuração
futura entra adicionando uma função no options.go.

E o lugar de cada tipo segue a regra das dependências: FileType mora no
domain porque a entidade usa ele. Tipo na camada errada o compilador
cobra com import cycle.