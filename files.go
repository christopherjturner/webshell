package main

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

const MAX_HOME_FILES int = 100

//go:embed templates/*
var templateFS embed.FS
var termTemplate = template.Must(template.ParseFS(templateFS, "templates/index.html"))
var fileTemplate = template.Must(template.ParseFS(templateFS, "templates/files.html"))

type termPageParams struct {
	Prefix   string
	ShellUrl string
}

func termPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shellUrl := path.Join(prefix, "shell")

	if err := termTemplate.Execute(w, termPageParams{ShellUrl: shellUrl, Prefix: prefix}); err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}
}

type filePageParams struct {
	HomeDir string
	Files   []string
}

func homeDirHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Add("Content-Type", "text/html")
	filePaths := []string{}

	filepath.Walk(config.HomeDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && len(filePaths) < MAX_HOME_FILES {
			relativePath, _ := filepath.Rel(config.HomeDir, path)
			filePaths = append(filePaths, relativePath)
		}
		return nil
	})

	if err := fileTemplate.Execute(w, filePageParams{HomeDir: config.HomeDir, Files: filePaths}); err != nil {
		log.Println(err)
	}
}

func getFileHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// sanitize input
	filename := filepath.Clean(r.PathValue("filename"))
	filename = filepath.Join(config.HomeDir, filename)
	println(filename)
	f, err := os.Open(filename)
	if err != nil {
		http.Error(w, "File Not Found", 404)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := io.Copy(w, f); err != nil {
		http.Error(w, "Error reading file", 500)
	}

	return
}

func uploadFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse multipart message", http.StatusInternalServerError)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filePath := filepath.Join(config.HomeDir, header.Filename)

	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "File uploaded successfully: %s", header.Filename)
}
