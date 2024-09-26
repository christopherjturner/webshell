package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/websocket"
)

//go:embed assets/*
var assetsFS embed.FS

var config Config

func main() {

	// DEBUG: List all the assets baked into the binary
	fs.WalkDir(assetsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(path)
		return nil
	})

	config = LoadConfigFromEnv()
	webshellMux := http.NewServeMux()

	// Add middleware to websocket handler
	var wsHandler http.Handler = websocket.Handler(shellHandler)

	// Middleware: Single use token
	if config.Once {
		wsHandler = haltOnExit(once(wsHandler))
		log.Println("Server will EXIT after the first connection closes")
	}

	// webshell routes
	webshellMux.HandleFunc("/{$}", termPageHandler)
	webshellMux.Handle("/shell", wsHandler)
	webshellMux.HandleFunc("/home", homeDirHandler)
	webshellMux.HandleFunc("/upload", uploadFileHandler)
	webshellMux.HandleFunc("/home/{filename...}", getFileHandler)
	webshellMux.Handle("/assets/", http.FileServer(http.FS(assetsFS)))

	// system routes
	systemMux := http.NewServeMux()
	systemMux.HandleFunc("/health", healthHandler)

	// combined routes
	rootMux := http.NewServeMux()

	if config.Token == "" {
		rootMux.Handle("/", webshellMux)
	} else {
		rootMux.Handle("/"+config.Token+"/", http.StripPrefix("/"+config.Token, webshellMux))
	}
	rootMux.Handle("/", systemMux)

	// start the server
	log.Printf("Listening on 0.0.0.0:%d", config.Port)
	log.Printf("Service files form %s", config.HomeDir)

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", config.Port),
		Handler:        debug(rootMux),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}

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

// Has the login token already been used
var keyUsed = false

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
