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
	"sync"
	"syscall"
	"time"
	"webshell/logging"

	"golang.org/x/net/websocket"
)

//go:embed assets/*
var assetsFS embed.FS

var (
	config      Config
	logger      *slog.Logger
	auditLogger *slog.Logger

	globalCtx         context.Context
	cancelFunc        context.CancelFunc
	activeConnections sync.WaitGroup
)

func main() {

	globalCtx, cancelFunc = context.WithCancel(context.Background())

	config = LoadConfigFromEnv()
	logger = logging.NewEcsLogger("terminal", config.LogLevel)
	auditLogger = logging.NewEcsLogger("session", config.LogLevel)

	routes := buildRoutes()

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

		shutdownCtx, shutdownRelease := context.WithTimeout(globalCtx, 10*time.Second)
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
	cancelFunc()
	activeConnections.Wait()
	logger.Info("All connections closed")
}

// Minimal healthcheck endpoint.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

func buildRoutes() http.Handler {

	rootPrefix := "/" + config.Token
	rootPath := rootPrefix + "/"
	var timeout Timeout = &NoOpTimeout{}
	if config.Once {
		timeout = NewInactivityTimeout(globalCtx, config.Grace)
	}

	var (
		wsHandler       http.Handler = Shell{config, timeout}
		termPageHandler http.Handler = termPageHandler(config.Token, config.Title)
		filesHandler    http.Handler = FilesHandler{
			baseDir: config.HomeDir,
			baseUrl: rootPath + "home",
			user:    config.User,
			logger:  logger,
		}.Handler()
		themeHandler = ThemeHandler{
			themeFile: config.Theme,
		}
	)

	// Add middleware to websocket handler.
	if config.Once {
		timeout = NewInactivityTimeout(globalCtx, config.Grace)
		o := NewOnceMiddleware(rootPrefix)
		wsHandler = o.once(wsHandler)
		termPageHandler = o.setCookie(termPageHandler)
		filesHandler = o.requireCookie(filesHandler)
		logger.Info("Server will EXIT after the first connection closes")
	}

	// Webshell routes.
	webshellMux := http.NewServeMux()
	webshellMux.Handle("/{$}", termPageHandler)
	webshellMux.Handle("/shell", wsHandler)
	webshellMux.Handle("/home", filesHandler)
	webshellMux.Handle("/upload", filesHandler)
	webshellMux.Handle("/home/{filename...}", filesHandler)
	webshellMux.Handle("/theme", themeHandler)
	webshellMux.Handle("/assets/", http.FileServer(http.FS(assetsFS)))

	// Playback of audit files. Still a work in progress
	if config.Replay {
		webshellMux.Handle("/replay/ws", websocket.Handler(replayHandler))
		webshellMux.Handle("/replay", replayPageHandler(config.Token))
	}

	// Combined routes.
	rootMux := http.NewServeMux()
	rootMux.Handle(rootPath, http.StripPrefix(rootPrefix, webshellMux))
	rootMux.HandleFunc("/health", healthHandler)
	rootMux.HandleFunc("/debug", debugHandler)

	return rootMux
}
