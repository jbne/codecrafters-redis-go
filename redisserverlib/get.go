package redisserverlib

import (
	"context"
	"fmt"
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

	if entry, exists := r.dataStore.Get(key); exists {
		if value, ok := entry.(redisType_String); ok {
			return fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)
		}

		return fmt.Sprintf("-ERR GET can only be called on string values! %s", c.getUsage(ctx))
	}

	return "$-1\r\n"
}
