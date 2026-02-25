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
	ClientIdKey  = "client_id"
	RequestIdKey = "request_id"
)

func NewHandler() *MyHandler {
	// Configure colored logging with tint
	return &MyHandler{
		Handler: tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: "2006-01-02 15:04:05.000",
			NoColor:    false,
			AddSource:  true,
		}),
	}
}

func (h *MyHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := ctx.Value(ClientIdKey).(int); ok {
		r.AddAttrs(slog.Int(ClientIdKey, id))
	}

	if id, ok := ctx.Value(RequestIdKey).(int); ok {
		r.AddAttrs(slog.Int(RequestIdKey, id))
	}

	return h.Handler.Handle(ctx, r)
}
