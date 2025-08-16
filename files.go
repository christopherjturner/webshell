package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type FileLink struct {
	Link  string
	Name  string
	IsDir bool
}

type fileParams struct {
	AssetsPath string
	UploadPath string
	CurrentDir string
	Error      string
	Files      []FileLink
}

type errorParams struct {
	AssetsPath string
	Home       string
	Error      string
}

type FilesHandler struct {
	baseDir string
	baseUrl string
	user    *user.User
	logger  *slog.Logger
}

func (fh FilesHandler) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Method == "POST" {
			fh.uploadFileHandler(w, r)
			return
		}

		// Path to the requested file on the local filesystem.
		filename := filepath.Join(fh.baseDir, filepath.Clean(r.PathValue("filename")))

		// Deny requests to anything outside of the baseDir.
		if !fh.isPathSafe(filename) {
			fh.logger.Error("Refusing to serve files from " + filename + " as its outside the root dir " + fh.baseDir)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Check file/dir exists.
		stat, err := os.Stat(filename)
		if err != nil {
			http.Error(w, "File Not Found", 404)
			return
		}

		if stat.IsDir() {
			fh.listFiles(w, filename, "")
		} else {
			fh.downloadFile(w, filename)
		}
	})
}

func (fh FilesHandler) downloadFile(w http.ResponseWriter, filename string) {

	f, err := os.Open(filename)
	if err != nil {
		fh.logger.Error(err.Error())
		http.Error(w, "File Not Found", http.StatusNotFound)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := io.Copy(w, f); err != nil {
		http.Error(w, "Error reading file "+f.Name(), http.StatusInternalServerError)
	}
}

func (fh FilesHandler) listFiles(w http.ResponseWriter, dirname string, error string) {

	fh.logger.Info(fmt.Sprintf("listing files in %s", dirname))

	params := fileParams{
		AssetsPath: path.Join(fh.baseUrl, "../assets"),
		UploadPath: path.Join(fh.baseUrl, "/upload"),
		CurrentDir: dirname,
		Files:      []FileLink{},
		Error:      error,
	}

	w.Header().Add("Content-Type", "text/html")

	files, err := os.ReadDir(dirname)
	if err != nil {
		params.Error = "Unable to list directory"
		if err := fileTemplate.Execute(w, params); err != nil {
			fh.logger.Error(fmt.Sprintf("%s", err))
		}
		return
	}

	// if in a subdir, link back to parent directory
	parent := filepath.Dir(dirname)

	if dirname != filepath.Clean(fh.baseDir) {
		params.Files = append(params.Files, FileLink{
			Name:  "..",
			Link:  fh.asLink(parent),
			IsDir: true,
		})
	}

	// List all the files.
	for _, file := range files {
		link := FileLink{
			Name:  file.Name(),
			Link:  fh.asLink(path.Join(dirname, file.Name())),
			IsDir: file.IsDir(),
		}
		params.Files = append(params.Files, link)
	}

	// Sort by dir first then by name.
	sort.Slice(params.Files, func(i, j int) bool {
		// directories come first
		if params.Files[i].IsDir != params.Files[j].IsDir {
			return params.Files[i].IsDir
		}
		// then sort by name
		return params.Files[i].Name < params.Files[j].Name
	})

	// Render the template.
	if err := fileTemplate.Execute(w, params); err != nil {
		fh.logger.Error(fmt.Sprintf("%s", err))
	}
}

// Handles uploads by the container.
func (fh FilesHandler) uploadFileHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		fh.fileError(w, "Method not allowed")
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		logger.Error(fmt.Sprintf("Multipart upload failed: %s", err.Error()))
		fh.fileError(w, "Failed to parse multipart message")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		fh.fileError(w, "Upload failed, invalid file")
		return
	}
	defer file.Close()

	if err := fh.upload(file, header.Filename); err != nil {
		fh.fileError(w, err.Error())
		return
	}

	// Reload the file page
	http.Redirect(w, r, fh.baseUrl, http.StatusSeeOther)
}

func (fh FilesHandler) upload(file io.Reader, filename string) error {

	filePath := filepath.Join(fh.baseDir, filepath.Clean(filename))

	if !fh.isPathSafe(filePath) {
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
	if fh.user != nil {
		if err := chown(dst, fh.user); err != nil {
			return errors.New("upload failed, failed to create file for user")
		}
	}

	if _, err := io.Copy(dst, file); err != nil {
		return errors.New("upload failed, failed to write file")
	}

	return nil
}

func (fh FilesHandler) fileError(w http.ResponseWriter, error string) {
	params := errorParams{
		AssetsPath: path.Join(fh.baseUrl, "../assets"),
		Home:       fh.baseUrl,
		Error:      error,
	}

	w.WriteHeader(http.StatusInternalServerError) // TODO: parameterize code
	if err := errorTemplate.Execute(w, params); err != nil {
		fh.logger.Error(fmt.Sprintf("%s", err))
	}
}

func (fh FilesHandler) isPathSafe(filename string) bool {
	return strings.HasPrefix(filepath.Clean(filename), filepath.Clean(fh.baseDir))
}

// Given an absolute path on the filesystem, return a url path
func (fh FilesHandler) asLink(filename string) string {
	// remove the basedir
	rel, err := filepath.Rel(fh.baseDir, filename)
	if err != nil {
		return ""
	}

	return filepath.Join(fh.baseUrl, rel)
}
