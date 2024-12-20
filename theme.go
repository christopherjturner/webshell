package main

import (
	"io"
	"net/http"
	"os"
)

type ThemeHandler struct {
	themeFile string
}

func (t ThemeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/javascript")
	if t.themeFile == "" {
		w.WriteHeader(200)
		return
	}

	f, err := os.Open(t.themeFile)
	if err != nil {
		logger.Info("theme not found")
		w.Header().Set("Content-Type", "text/javascript")
		w.WriteHeader(200)
		return
	}
	defer f.Close()

	if _, err := io.Copy(w, f); err != nil {
		http.Error(w, "Error reading theme file "+f.Name(), http.StatusInternalServerError)
	}

}
