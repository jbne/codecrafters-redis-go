package redisserverlib

import (
	"context"
	"fmt"
	"log/slog"
)

type (
	get struct{}
)

func (c get) getUsage(ctx context.Context) string {
	return `
usage:
	get key
summary:
	Get the value of key. If the key does not exist the special value nil is returned.
	An error is returned if the value stored at key is not a string, because GET only handles string values.
` + "\r\n"
}

func (c get) execute(ctx context.Context, r *redisCommandProcessor, params commandParams) commandResult {
	if len(params) < 2 {
		return "-ERR No key provided to GET!\r\n"
	}
	key := params[1]
	response, ok := r.cache.Get(key)

	if ok {
		slog.DebugContext(ctx, "GET cache hit", "key", key)
		return fmt.Sprintf("$%d\r\n%s\r\n", len(response), response)
	}

	slog.DebugContext(ctx, "GET cache miss", "key", key)
	return "$-1\r\n"
}
