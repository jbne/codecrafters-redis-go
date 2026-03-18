package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

type (
	MyHandler struct {
		slog.Handler
	}
)

const (
	ClientKey    = "client"
	RequestIdKey = "request_id"
)

func NewHandler(logLevel slog.Level) *MyHandler {
	// Configure colored logging with tint
	return &MyHandler{
		Handler: tint.NewHandler(os.Stderr, &tint.Options{
			Level:      logLevel,
			TimeFormat: "2006-01-02 15:04:05.000",
			NoColor:    false,
			AddSource:  true,
		}),
	}
}

func (h *MyHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := ctx.Value(ClientKey).(string); ok {
		r.AddAttrs(slog.String(ClientKey, id))
	}

	if id, ok := ctx.Value(RequestIdKey).(int); ok {
		r.AddAttrs(slog.Int(RequestIdKey, id))
	}

	return h.Handler.Handle(ctx, r)
}
