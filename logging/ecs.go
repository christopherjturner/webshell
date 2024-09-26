package logging

import (
	"context"
	"io"
	"log/slog"
)

const (
	ecsVersion = "8.10.0"
	logger     = "log/slog"
)

const (
	ecsVersionKey = "ecs.version"

	timestampKey = "@timestamp"
	messageKey   = "message"
	logLevelKey  = "log.level"
	logLoggerKey = "log.logger"
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
	jsonHandler slog.Handler
}

func NewHandler(w io.Writer, logLevel *slog.LevelVar) *Handler {
	return &Handler{
		jsonHandler: slog.NewJSONHandler(w, &slog.HandlerOptions{Level: logLevel, ReplaceAttr: replacer}),
	}
}

func (x *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return x.jsonHandler.Enabled(ctx, level)
}

func (x *Handler) Handle(ctx context.Context, record slog.Record) error {

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
		slog.Time(timestampKey, record.Time),
		slog.String(messageKey, record.Message),
		slog.String(logLevelKey, level),
		slog.String(ecsVersionKey, ecsVersion),
	)
	return x.jsonHandler.Handle(ctx, record)
}

func (x *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{jsonHandler: x.jsonHandler.WithAttrs(attrs)}
}

func (x *Handler) WithGroup(name string) slog.Handler {
	return &Handler{jsonHandler: x.jsonHandler.WithGroup(name)}
}

