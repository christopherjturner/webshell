package main

import (
	"embed"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"
	"webshell/logging"

	"golang.org/x/net/websocket"
)

//go:embed assets/*
var assetsFS embed.FS
var config Config
var logger *slog.Logger
var auditLogger *slog.Logger

func main() {

	config = LoadConfigFromEnv()

	// Set up loggers.
	logger = logging.NewEcsLogger("terminal", config.LogLevel)
	auditLogger = logging.NewEcsLogger("session", config.LogLevel)

	// Add middleware to websocket handler.
	var wsHandler http.Handler = websocket.Handler(shellHandler)
	if config.Once {
		wsHandler = haltOnExit(once(wsHandler))
		logger.Info("Server will EXIT after the first connection closes")
	}

	// Webshell routes.
	webshellMux := http.NewServeMux()
	webshellMux.HandleFunc("/{$}", termPageHandler)
	webshellMux.Handle("/shell", wsHandler)

	// Playback of audit files. Still a work in progress
	if config.Replay {
		wsReplayHandler := websocket.Handler(replayHandler)
		webshellMux.Handle("/replay/ws", wsReplayHandler)
		webshellMux.HandleFunc("/replay", replayPageHandler)
	}

	webshellMux.HandleFunc("/home", getFileHandler)
	webshellMux.HandleFunc("/upload", uploadFileHandler)
	webshellMux.HandleFunc("/home/{filename...}", getFileHandler)
	webshellMux.Handle("/assets/", http.FileServer(http.FS(assetsFS)))

	// Combined routes.
	rootMux := http.NewServeMux()
	rootMux.Handle("/"+config.Token+"/", http.StripPrefix("/"+config.Token, webshellMux))

	rootMux.HandleFunc("/health", healthHandler)

	logger.Info(fmt.Sprintf("Listening on 0.0.0.0:%d", config.Port))
	logger.Info(fmt.Sprintf("Service files form %s", config.HomeDir))

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", config.Port),
		Handler:        requestLogger(rootMux),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(s.ListenAndServe())
}

// Middleware to log inbound requests.
func requestLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info(r.URL.Path)
		h.ServeHTTP(w, r)
	})
}

// Middleware to stop the process after the webshell disconnects.
func haltOnExit(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
		os.Exit(0)
	})
}

// Has the login token already been used.
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

// Minimal healthcheck endpoint.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}
