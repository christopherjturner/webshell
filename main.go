package main

import (
	"embed"
	"fmt"
	"golang.org/x/net/websocket"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

//go:embed static/*
var assetsFS embed.FS

var config Config
var prefix string = ""

func main() {

	config = LoadConfigFromEnv()
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)

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

	mux.Handle(path.Join(prefix, "shell"), wsHandler)
	mux.HandleFunc(path.Join(prefix, "home"), homeDirHandler)
	mux.HandleFunc(path.Join(prefix, "upload"), uploadFileHandler)
	mux.HandleFunc(path.Join(prefix, "home/{filename...}"), getFileHandler)
	mux.HandleFunc(prefix+"/{$}", termPageHandler)

	assetPath := fmt.Sprintf("%s/assets/", prefix)
	log.Printf("asset path %s", assetPath)
	//staticAssets := http.StripPrefix(assetPath, http.FileServer(http.Dir("./static/")))
	staticAssets := http.StripPrefix(assetPath, http.FileServer(http.FS(assetsFS)))
	mux.Handle(assetPath, staticAssets)

	log.Printf("Listening on 0.0.0.0:%d/%s", config.Port, prefix)
	log.Printf("Service files form %s", config.HomeDir)

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", config.Port),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())

}

var keyUsed = false

func debugHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upath := r.URL.Path
		if !strings.HasPrefix(upath, "/") {
			upath = "/" + upath
			r.URL.Path = upath
		}
		log.Printf("%s serving file from %s", r.URL.Path, path.Clean(upath))

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
