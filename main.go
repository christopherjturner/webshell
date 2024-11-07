package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	ctx, cancel := context.WithCancel(context.Background())

	config = LoadConfigFromEnv()
	logger = logging.NewEcsLogger("terminal", config.LogLevel)
	auditLogger = logging.NewEcsLogger("session", config.LogLevel)

	routes := buildRoutes(ctx)

	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", config.Port),
		Handler:        requestLogger(routes),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Handle shutdown signals.
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		logger.Warn("Shutdown signal received")
		shutdownCtx, shutdownRelease := context.WithTimeout(ctx, 10*time.Second)
		defer shutdownRelease()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("HTTP shutdown error: %v", err)
		}
	}()

	// Start the server.
	logger.Info(fmt.Sprintf("Listening on 0.0.0.0:%d", config.Port))
	logger.Info(fmt.Sprintf("Audit TTY  %t", config.AuditTTY))
	logger.Info(fmt.Sprintf("Audit Exec %t", config.AuditExec))

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}

	// Gracefully stop any active websocket connections.
	logger.Info("Waiting for handlers to exit")
	cancel()
	ttyWait.Wait()
}

func withCtx(h http.Handler, ctx context.Context) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Middleware to log inbound requests.
func requestLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info(r.URL.Path)
		h.ServeHTTP(w, r)
	})
}

// Has the login token already been used.
var keyUsed = false

// Allow only one connection and stop the server when its closed.
func once(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if keyUsed {
			http.Error(w, "expired", http.StatusUnauthorized)
			return
		}
		keyUsed = true
		h.ServeHTTP(w, r)
		os.Exit(0)
	})
}

// Minimal healthcheck endpoint.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

func buildRoutes(ctx context.Context) http.Handler {
	// Add middleware to websocket handler.
	var wsHandler http.Handler = websocket.Handler(shellHandler)
	if config.Once {
		wsHandler = once(withCtx(wsHandler, ctx))
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
	rootMux.HandleFunc("/debug", debugHandler)

	return rootMux
}
