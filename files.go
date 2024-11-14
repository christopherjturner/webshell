package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
)

type FileLink struct {
	Path  string
	Name  string
	IsDir bool
}

type filePageParams struct {
	AssetsPath string
	UploadPath string
	CurrentDir string
	Error      string
	Files      []FileLink
}

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
		logger.Error("%v\n", err)
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

	params := filePageParams{
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

	if err := upload(file, header.Filename); err != nil {
		fileError(w, err.Error())
		return
	}

	// Reload the file page
	homePath := path.Join("/", config.Token, "/home")
	http.Redirect(w, r, homePath, http.StatusSeeOther)
}

func upload(file io.Reader, filename string) error {

	filePath := filepath.Join(config.HomeDir, filepath.Clean(filename))

	if !isPathSafe(filePath) {
		return errors.New("file is outside of the homedir")
	}

	// Make sure we don't overwrite anything.
	if checkFileExists(filePath) {
		return errors.New("file already exists")
	}

	dst, err := os.Create(filePath)
	if err != nil {
		return errors.New("upload failed, failed to create file")
	}
	defer dst.Close()

	// Ensure the file is owned by the shell user, not the server
	if config.User != nil {
		if err := chown(dst, config.User); err != nil {
			return errors.New("upload failed, failed to create file for user")
		}
	}

	if _, err := io.Copy(dst, file); err != nil {
		return errors.New("upload failed, failed to write file")
	}

	return nil
}

func fileError(w http.ResponseWriter, error string) {
	listFiles(w, ".", error)
}
