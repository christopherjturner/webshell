package main

import (
	"embed"
	"fmt"
	"golang.org/x/net/websocket"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

//go:embed assets/*
var assetsFS embed.FS

var config Config
var prefix string = ""

func main() {

	fs.WalkDir(assetsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(path)
		return nil
	})

	config = LoadConfigFromEnv()
	mux := http.NewServeMux()

	var wsHandler http.Handler = websocket.Handler(shellHandler)
	if config.Token != "" {
		log.Printf("TOKEN %s", config.Token)
		//wsHandler = checkKey(config.Token, wsHandler)
		prefix = path.Join("/", url.PathEscape(config.Token))
	} else {
		log.Println("No access token set! Anyone with the url can access the shell!")
	}

	if config.Once {
		wsHandler = haltOnExit(once(wsHandler))
		log.Println("Server will EXIT after the first connection closes")
	}

	mux.HandleFunc(prefix+"/{$}", termPageHandler)
	mux.Handle(path.Join(prefix, "shell"), wsHandler)
	mux.HandleFunc(path.Join(prefix, "home"), homeDirHandler)
	mux.HandleFunc(path.Join(prefix, "upload"), uploadFileHandler)
	mux.HandleFunc(path.Join(prefix, "home/{filename...}"), getFileHandler)

	assetPath := fmt.Sprintf("%s/assets/", prefix)
	staticAssets := http.StripPrefix(prefix, http.FileServer(http.FS(assetsFS)))
	mux.Handle(assetPath, staticAssets)

	mux.HandleFunc("/health", healthHandler)

	log.Printf("Listening on 0.0.0.0:%d/%s", config.Port, prefix)
	log.Printf("Service files form %s", config.HomeDir)

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", config.Port),
		Handler:        debug(mux),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())

}

var keyUsed = false

func debug(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s", r.URL.Path)

		h.ServeHTTP(w, r)
	})
}

func haltOnExit(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
		os.Exit(0)
	})
}

func once(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if keyUsed {
			http.Error(w, "expired", http.StatusUnauthorized)
			return
		}
		keyUsed = true
		h.ServeHTTP(w, r)
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}
