package main

import (
	"net/http"

	"github.com/fabiobap/go-pdf-optimizer/internal/config"
	"github.com/fabiobap/go-pdf-optimizer/internal/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func routes(app *config.AppConfig) http.Handler {
	mux := chi.NewRouter()
	mux.Use(middleware.Recoverer)
	mux.Use(NoSurf)
	mux.Use(SessionLoad)

	fileServer := http.FileServer(http.Dir("./static/"))
	mux.Handle("/static/*", http.StripPrefix("/static", fileServer))
	mux.Handle("/temp-pdf/", http.StripPrefix("/temp-pdf/", http.FileServer(http.Dir("temp-pdf"))))

	mux.Get("/", handlers.Repo.Home)
	mux.Get("/pdf-optimizer", handlers.Repo.PDFOptimizer)
	mux.Post("/pdf-optimizer", handlers.Repo.PostPDFOptimizer)
	mux.Get("/pdf-split", handlers.Repo.PDFSplit)
	mux.Post("/pdf-split", handlers.Repo.PostPDFSplit)

	return mux
}
