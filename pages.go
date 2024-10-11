package main

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
)

//go:embed templates/*
var templateFS embed.FS

// As with assets, templates are embedded in the binary.
var (
	termTemplate = template.Must(template.ParseFS(templateFS, "templates/index.html"))
	fileTemplate = template.Must(template.ParseFS(templateFS, "templates/files.html"))
)

type termPageParams struct {
	Token string
}

type FileLink struct {
	Path  string
	Name  string
	IsDir bool
}

type homeDirParams struct {
	AssetsPath string
	UploadPath string
	CurrentDir string
	Error      string
	Files      []FileLink
}

// Handles rendering the main xterm.js page.
func termPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := termTemplate.Execute(w, termPageParams{Token: config.Token}); err != nil {
		logger.Error(fmt.Sprintf("%s", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// Handler for displaying directory content and downloading files.
func getFileHandler(w http.ResponseWriter, r *http.Request) {

	filename := filepath.Clean(r.PathValue("filename"))

	stat, err := os.Stat(path.Join(config.HomeDir, filename))
	if err != nil {
		http.Error(w, "File Not Found", 404)
		return
	}

	if stat.IsDir() {
		listFiles(w, filename, "")
	} else {
		downloadFile(w, filename)
	}
}

// Handler for downloading a given file.
func downloadFile(w http.ResponseWriter, filename string) {
	f, err := os.Open(path.Join(config.HomeDir, filename))
	if err != nil {
		http.Error(w, "File Not Found", 404)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := io.Copy(w, f); err != nil {
		http.Error(w, "Error reading file "+f.Name(), 500)
	}
}

// Handles listing the contents of a directory.
func listFiles(w http.ResponseWriter, filename string, error string) {

	homePath := path.Join("/", config.Token, "/home")

	params := homeDirParams{
		AssetsPath: path.Join("/", config.Token, "/assets"),
		UploadPath: path.Join("/", config.Token, "/upload"),
		CurrentDir: path.Join(config.HomeDir, filename),
		Files:      []FileLink{},
		Error:      error,
	}

	w.Header().Add("Content-Type", "text/html")

	f, err := os.Open(path.Join(config.HomeDir, filename))
	if err != nil {
		params.Error = "Unable read directory"
		if err := fileTemplate.Execute(w, params); err != nil {
			logger.Error(fmt.Sprintf("%s", err))
		}
		return
	}
	defer f.Close()

	files, err := f.Readdir(0)
	if err != nil {
		params.Error = "Unable to list directory"
		if err := fileTemplate.Execute(w, params); err != nil {
			logger.Error(fmt.Sprintf("%s", err))
		}
		return
	}

	// Link back to parent dir.
	if filename != "." {
		params.Files = append(params.Files, FileLink{
			Name:  "..",
			Path:  path.Clean(path.Join(homePath, filename, "..")),
			IsDir: true,
		})
	}

	// List all the files.
	for _, file := range files {
		link := FileLink{
			Name:  file.Name(),
			Path:  path.Clean(path.Join(homePath, filename, file.Name())),
			IsDir: file.IsDir(),
		}
		params.Files = append(params.Files, link)
	}

	// Sort by dir first then by name.
	sort.Slice(params.Files, func(i, j int) bool {
		if params.Files[i].IsDir && !params.Files[j].IsDir {
			return true
		}

		return (params.Files[i].IsDir || params.Files[j].IsDir) &&
			params.Files[i].Name < params.Files[j].Name
	})

	// Render the template.
	w.Header().Add("Content-Type", "text/html")
	if err := fileTemplate.Execute(w, params); err != nil {
		logger.Error(fmt.Sprintf("%s", err))
	}
}

// Handles uploads by the container.
func uploadFileHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate payload
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		listFiles(w, ".", "Failed to parse multipart message")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		listFiles(w, ".", "Upload failed, invalid file")
		return
	}
	defer file.Close()

	// TODO: upload file to the selected dir rather than home dir
	filePath := filepath.Join(config.HomeDir, header.Filename)

	dst, err := os.Create(filePath)
	if err != nil {
		listFiles(w, ".", "Upload failed, failed to create file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		listFiles(w, ".", "Upload failed, failed to write file")
		return
	}

	// Reload the file page
	homePath := path.Join("/", config.Token, "/home")
	http.Redirect(w, r, homePath, http.StatusSeeOther)
}
