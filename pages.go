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
	termTemplate   = template.Must(template.ParseFS(templateFS, "templates/index.html"))
	replayTemplate = template.Must(template.ParseFS(templateFS, "templates/replay.html"))
	fileTemplate   = template.Must(template.ParseFS(templateFS, "templates/files.html"))
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

// Xterm.js handlers

func termPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Printf("Token is being set to %s\n", config.Token)
	if err := termTemplate.Execute(w, termPageParams{Token: config.Token}); err != nil {
		logger.Error(fmt.Sprintf("%s", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func replayPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := replayTemplate.Execute(w, termPageParams{Token: config.Token}); err != nil {
		logger.Error(fmt.Sprintf("%s", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// File upload handlers

func getFileHandler(w http.ResponseWriter, r *http.Request) {

	filename := filepath.Clean(r.PathValue("filename"))

	stat, err := os.Stat(filepath.Join(config.HomeDir, filename))
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

func downloadFile(w http.ResponseWriter, filename string) {

	filename = filepath.Join(config.HomeDir, filepath.Clean(filename))
	logger.Info(fmt.Sprintf("Downloading %s", filename))

	if !isPathSafe(filename) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("%v\n", err)
		http.Error(w, "File Not Found", http.StatusNotFound)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := io.Copy(w, f); err != nil {
		http.Error(w, "Error reading file "+f.Name(), http.StatusInternalServerError)
	}
}

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

	dirToList := filepath.Join(config.HomeDir, filepath.Clean(filename))

	if !isPathSafe(dirToList) {
		logger.Error(fmt.Sprintf("Declined to list files in %s", dirToList))
		params.Error = "Unable to list directory outside of home dir."
		return
	}

	f, err := os.Open(dirToList)
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
			Path:  path.Join(homePath, filename, file.Name()),
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
		fileError(w, "Method not allowed")
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		fileError(w, "Failed to parse multipart message")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		fileError(w, "Upload failed, invalid file")
		return
	}
	defer file.Close()

	filePath := filepath.Join(config.HomeDir, filepath.Clean(header.Filename))

	if !isPathSafe(filePath) {
		fileError(w, "File is outside of the homedir")
		return
	}

	// Make sure we don't overwrite anything.
	if checkFileExists(filePath) {
		fileError(w, "File already exists")
		return
	}

	dst, err := os.Create(filePath)
	if err != nil {
		fileError(w, "Upload failed, failed to create file")
		return
	}
	defer dst.Close()

	// Ensure the file is owned by the shell user, not the server
	if config.User != nil {
		if err := chown(dst, config.User); err != nil {
			fileError(w, "Upload failed, failed to create file for user")
			return
		}
	}

	if _, err := io.Copy(dst, file); err != nil {
		fileError(w, "Upload failed, failed to write file")
		return
	}

	// Reload the file page
	homePath := path.Join("/", config.Token, "/home")
	http.Redirect(w, r, homePath, http.StatusSeeOther)
}

func fileError(w http.ResponseWriter, error string) {
	listFiles(w, ".", error)
}
