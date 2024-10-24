package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
)

const (
	logger        = "log/slog"
	ecsVersion    = "8.10.0"
	ecsVersionKey = "ecs.version"
	timestampKey  = "@timestamp"
	messageKey    = "message"
	logLevelKey   = "log.level"
	logLoggerKey  = "log.logger"
	logKindKey    = "log.kind"
	sourceKey     = "webshell.source"
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
		slog.String(sourceKey, x.source),
	)
	return x.jsonHandler.Handle(ctx, record)
}

func (x *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{jsonHandler: x.jsonHandler.WithAttrs(attrs)}
}

func (x *Handler) WithGroup(name string) slog.Handler {
	return &Handler{jsonHandler: x.jsonHandler.WithGroup(name)}
}

func NewEcsLogger(source string, logLevel *slog.LevelVar) *slog.Logger {
	return slog.New(NewHandler(os.Stdout, source, logLevel))
}
