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
var routes Routes

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

	// Add middleware to websocket handler
	var wsHandler http.Handler = websocket.Handler(shellHandler)
	if config.Token != "" {
		log.Printf("TOKEN %s", config.Token)
		//wsHandler = checkKey(config.Token, wsHandler)
	} else {
		log.Println("No access token set! Anyone with the url can access the shell!")
	}

	if config.Once {
		wsHandler = haltOnExit(once(wsHandler))
		log.Println("Server will EXIT after the first connection closes")
	}

	// routes
	routes = BuildRoutes(config.Token)

	mux.HandleFunc(routes.Main, termPageHandler)
	mux.Handle(routes.Shell, wsHandler)
	mux.HandleFunc(routes.Home, homeDirHandler)
	mux.HandleFunc(routes.Upload, uploadFileHandler)
	mux.HandleFunc(routes.GetFile, getFileHandler)

	staticAssets := http.StripPrefix(routes.Prefix, http.FileServer(http.FS(assetsFS)))
	mux.Handle(routes.Assets, staticAssets)

	mux.HandleFunc("/health", healthHandler)

	log.Printf("Listening on 0.0.0.0:%d/%s", config.Port, routes.Prefix)
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
