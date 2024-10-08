package handlers

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fabiobap/go-pdf-optimizer/internal/config"
	"github.com/fabiobap/go-pdf-optimizer/internal/models"
	"github.com/fabiobap/go-pdf-optimizer/internal/render"
)

var Repo *Repository

type Repository struct {
	App *config.AppConfig
}

func NewRepo(ac *config.AppConfig) *Repository {
	return &Repository{
		App: ac,
	}
}

func NewHandlers(r *Repository) {
	Repo = r
}

func (m *Repository) Home(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "home.page.tmpl", &models.TemplateData{})
}

func (m *Repository) PDFOptimizer(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "pdf-optimizer.page.tmpl", &models.TemplateData{})
}

func (m *Repository) PostPDFOptimizer(w http.ResponseWriter, r *http.Request) {
	file, handler, err := r.FormFile("pdfFile")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check if the file is a PDF
	if filepath.Ext(handler.Filename) != ".pdf" {
		http.Error(w, "Only PDF files are allowed", http.StatusBadRequest)
		return
	}

	// Create a temporary file within our temp-pdf directory
	tempFile, err := os.Create(filepath.Join("./temp-pdf", handler.Filename))
	if err != nil {
		log.Printf("Error saving file: %v", err)
		http.Error(w, "Unable to create the file", http.StatusInternalServerError)
		return
	}

	// Copy the uploaded file to the created file on the filesystem
	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "Unable to save the file", http.StatusInternalServerError)
		return
	}

	// Optimize the PDF using pdfcpu
	optimizedFilePath := filepath.Join("temp-pdf", "optimized_"+handler.Filename)
	cmd := exec.Command("pdfcpu", "optimize", tempFile.Name(), optimizedFilePath)
	err = cmd.Run()
	if err != nil {
		http.Error(w, "Unable to optimize the PDF file", http.StatusInternalServerError)
		return
	}

	defer tempFile.Close()

	// Send the optimized PDF back to the client
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(optimizedFilePath))
	w.Header().Set("Content-Type", "application/pdf")
	http.ServeFile(w, r, optimizedFilePath)
}

func (m *Repository) PDFSplit(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "pdf-split.page.tmpl", &models.TemplateData{})
}

func (m *Repository) PostPDFSplit(w http.ResponseWriter, r *http.Request) {
}
