package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
)

const (
	ecsVersion = "8.10.0"
)

func replacer(groups []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case "time", "msg", "source", "level":
		return slog.Attr{}
	default:
		return a
	}
}

type Handler struct {
	source      string
	jsonHandler slog.Handler
}

func NewHandler(w io.Writer, source string, logLevel *slog.LevelVar) *Handler {
	return &Handler{
		source:      source,
		jsonHandler: slog.NewJSONHandler(w, &slog.HandlerOptions{Level: logLevel, ReplaceAttr: replacer}),
	}
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.jsonHandler.Enabled(ctx, level)
}

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {

	var level = "info"
	switch record.Level {
	case slog.LevelDebug:
		level = "debug"
	case slog.LevelInfo:
		level = "info"
	case slog.LevelWarn:
		level = "warn"
	case slog.LevelError:
		level = "error"
	}

	record.AddAttrs(
		slog.String("ecs.version", ecsVersion),
		slog.Time("@timestamp", record.Time),
		slog.String("message", record.Message),
		slog.String("log.level", level),
		slog.Group("user",
			slog.String("id", os.Getenv("USER_ID")),
			slog.String("name", os.Getenv("USER_NAME")),
		),
		slog.Group("webshell",
			slog.String("source", h.source),
			slog.String("token", os.Getenv("TOKEN")),
			slog.String("service", os.Getenv("SERVICE")),
		),
	)

	return h.jsonHandler.Handle(ctx, record)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{jsonHandler: h.jsonHandler.WithAttrs(attrs)}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{jsonHandler: h.jsonHandler.WithGroup(name)}
}

func NewEcsLogger(source string, logLevel *slog.LevelVar) *slog.Logger {
	return slog.New(NewHandler(os.Stdout, source, logLevel))
}
