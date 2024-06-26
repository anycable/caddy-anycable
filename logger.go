package caddy_anycable

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type CaddyLogHandler struct {
	logger *zap.Logger
}

func NewCaddyLogHandler() *CaddyLogHandler {
	return &CaddyLogHandler{
		logger: caddy.Log(),
	}
}

func (h *CaddyLogHandler) Enabled(_ context.Context, level slog.Level) bool {
	zapLevel := zapLevelFromSlogLevel(level)

	return h.logger.Core().Enabled(zapLevel)
}

func (h *CaddyLogHandler) Handle(_ context.Context, record slog.Record) error {
	level := zapLevelFromSlogLevel(record.Level)
	msg := record.Message

	record.Attrs(func(attr slog.Attr) bool {
		msg += fmt.Sprintf(" %s=%s", attr.Key, attr.Value.String())
		return true
	})

	h.logger.Log(level, msg)
	return nil
}

func (h *CaddyLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newLogger := h.logger.With(zapFieldsFromAttrs(attrs)...)
	return &CaddyLogHandler{logger: newLogger}
}

func (h *CaddyLogHandler) WithGroup(name string) slog.Handler {
	newLogger := h.logger.Named(name)
	return &CaddyLogHandler{logger: newLogger}
}

func zapFieldsFromAttrs(attrs []slog.Attr) []zap.Field {
	fields := make([]zap.Field, len(attrs))
	for i, attr := range attrs {
		fields[i] = zap.Any(attr.Key, attr.Value)
	}
	return fields
}

func zapLevelFromSlogLevel(level slog.Level) zapcore.Level {
	var zapLevel zapcore.Level

	switch level {
	case slog.LevelDebug:
		zapLevel = zap.DebugLevel
	case slog.LevelInfo:
		zapLevel = zap.InfoLevel
	case slog.LevelWarn:
		zapLevel = zap.WarnLevel
	case slog.LevelError:
		zapLevel = zap.ErrorLevel
	default:
		zapLevel = zap.InfoLevel
	}

	return zapLevel
}
