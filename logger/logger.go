package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

type LoggerType struct {
	*slog.Logger
}

var logger LoggerType

func init() {
	// Configure colored logging with tint
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: "2006-01-02 15:04:05.000",
		NoColor:    false,
	})

	logger.Logger = slog.New(handler)
}

func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}

func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	logger.InfoContext(ctx, msg, args...)
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	logger.DebugContext(ctx, msg, args...)
}

func ErrorContext(ctx context.Context, msg string, args ...any) {
	logger.ErrorContext(ctx, msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	logger.WarnContext(ctx, msg, args...)
}