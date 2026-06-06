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

	livroEBook, err := service.Cadastrar(ctx, "Go in Action", "Ketelsen", application.MarcarLivroSendoDigital())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("created: %+v\n", *livroEBook)

	livroEBookPDF, err := service.Cadastrar(ctx, "Go in Action", "Ketelsen", application.MarcarTipo(domain.FileTypePDF))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("created: %+v\n", *livroEBookPDF)
}
