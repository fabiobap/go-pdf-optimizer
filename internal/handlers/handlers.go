package handlers

import (
	"archive/zip"
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/fabiobap/go-pdf-optimizer/internal/config"
	"github.com/fabiobap/go-pdf-optimizer/internal/models"
	"github.com/fabiobap/go-pdf-optimizer/internal/render"
	"github.com/pdfcpu/pdfcpu/pkg/api"
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
	// Parse the form
	err := r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Get the PDF file
	file, header, err := r.FormFile("pdfFile")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check if the file has a .pdf extension
	if filepath.Ext(header.Filename) != ".pdf" {
		http.Error(w, "File is not a PDF", http.StatusBadRequest)
		return
	}

	// Read the PDF file into a buffer
	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		http.Error(w, "Error reading the file", http.StatusInternalServerError)
		return
	}

	// Save the buffer to a temporary file
	tempFile, err := os.CreateTemp("", "uploaded-*.pdf")
	if err != nil {
		http.Error(w, "Error creating temporary file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())

	_, err = io.Copy(tempFile, &buf)
	if err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		return
	}

	outputFileName := "optimized.pdf"

	// Optimize the PDF
	err = api.OptimizeFile(tempFile.Name(), outputFileName, nil)
	if err != nil {
		http.Error(w, "Error optimizing PDF", http.StatusInternalServerError)
		log.Printf("Error optimizing PDF: %v", err)
		return
	}

	// Read the optimized PDF back into a buffer
	optimizedFile, err := os.Open(outputFileName)
	if err != nil {
		http.Error(w, "Error opening optimized file", http.StatusInternalServerError)
		return
	}
	defer optimizedFile.Close()

	var optimizedBuf bytes.Buffer
	_, err = io.Copy(&optimizedBuf, optimizedFile)
	if err != nil {
		http.Error(w, "Error reading optimized file", http.StatusInternalServerError)
		return
	}

	// Return the optimized PDF
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"compressed_"+header.Filename+"\"")
	w.Write(optimizedBuf.Bytes())
}

func (m *Repository) PDFSplit(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "pdf-split.page.tmpl", &models.TemplateData{})
}

func (m *Repository) PostPDFSplit(w http.ResponseWriter, r *http.Request) {
	//Parse the form
	err := r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "Internal error")
		log.Printf("Error parsing form: %v", err)
		http.Redirect(w, r, "/pdf-split", http.StatusSeeOther)
		return
	}

	// Retrieve the file from form data
	file, handler, err := r.FormFile("pdfFile")
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "Internal error")
		log.Printf("Error retrieving file: %v", err)
		http.Redirect(w, r, "/pdf-split", http.StatusSeeOther)
		return
	}
	defer file.Close()

	// Check if the file is a PDF
	if filepath.Ext(handler.Filename) != ".pdf" {
		m.App.Session.Put(r.Context(), "error", "Only PDF files are allowed")
		http.Redirect(w, r, "/pdf-split", http.StatusSeeOther)
		return
	}

	// Get the page_per_file value
	perPageStr := r.FormValue("page_per_file")
	perPage, err := strconv.Atoi(perPageStr)
	if err != nil || perPage < 1 {
		m.App.Session.Put(r.Context(), "error", "Page per field cannot be null nor less than 0")
		http.Redirect(w, r, "/pdf-split", http.StatusSeeOther)
		return
	}

	// Read the PDF file into a buffer
	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "Error reading the file")
		http.Redirect(w, r, "/pdf-split", http.StatusSeeOther)
		return
	}

	// Get the total number of pages in the PDF
	// workaorund because api.PageCount without reader was stuck
	reader := bytes.NewReader(buf.Bytes())
	totalPages, err := api.PageCount(reader, nil)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "Error getting total pages!")
		http.Redirect(w, r, "/pdf-split", http.StatusSeeOther)
		return
	}

	// Validate per_page against total pages
	if perPage > totalPages {
		m.App.Session.Put(r.Context(), "error", "Value of page per file exceeds total number of pages!")
		http.Redirect(w, r, "/pdf-split", http.StatusSeeOther)
	}

	// Save the buffer to a temporary file
	tempFile, err := os.CreateTemp("", "uploaded-*.pdf")
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "Internal error")
		log.Printf("Error creating temporary file: %v", err)
		http.Redirect(w, r, "/pdf-split", http.StatusSeeOther)
		return
	}
	defer os.Remove(tempFile.Name())

	_, err = io.Copy(tempFile, &buf)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "Internal error")
		log.Printf("Error saving the file: %v", err)
		http.Redirect(w, r, "/pdf-split", http.StatusSeeOther)
		return
	}

	// Split the PDF
	outputDir, err := os.MkdirTemp("", "split-")
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "Internal error")
		log.Printf("Error creating temporary directory: %v", err)
		http.Redirect(w, r, "/pdf-split", http.StatusSeeOther)
		return
	}
	defer os.RemoveAll(outputDir)

	err = api.SplitFile(tempFile.Name(), outputDir, perPage, nil)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "Error splitting PDF")
		http.Redirect(w, r, "/pdf-split", http.StatusSeeOther)
		return
	}

	// Create a zip file
	zipBuf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuf)

	files, err := os.ReadDir(outputDir)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "Internal error")
		log.Printf("Error reading split files: %v", err)
		http.Redirect(w, r, "/pdf-split", http.StatusSeeOther)
		return
	}

	for _, file := range files {
		f, err := zipWriter.Create(file.Name())
		if err != nil {
			m.App.Session.Put(r.Context(), "error", "Internal error")
			log.Printf("Error creating zip file: %v", err)
			http.Redirect(w, r, "/pdf-split", http.StatusSeeOther)
			return
		}

		partFile, err := os.Open(filepath.Join(outputDir, file.Name()))
		if err != nil {
			m.App.Session.Put(r.Context(), "error", "Internal error")
			log.Printf("Error opening split file: %v", err)
			http.Redirect(w, r, "/pdf-split", http.StatusSeeOther)
			return
		}
		defer partFile.Close()

		_, err = io.Copy(f, partFile)
		if err != nil {
			m.App.Session.Put(r.Context(), "error", "Internal error")
			log.Printf("Error writing to zip file: %v", err)
			http.Redirect(w, r, "/pdf-split", http.StatusSeeOther)
			return
		}
	}
	zipWriter.Close()

	// Return the zip file
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"split_pdfs.zip\"")
	w.Write(zipBuf.Bytes())
}
