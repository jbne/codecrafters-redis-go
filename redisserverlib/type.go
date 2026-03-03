package redisserverlib

import (
	"context"
	"fmt"
)

type (
	typeCmd struct{}
)

func (c typeCmd) getUsage(ctx context.Context) string {
	return `
usage:
	TYPE key
summary:
	Returns the string representation of the type of the value stored at key.
	The different types that can be returned are: string, list, set, zset, hash, stream, and vectorset.
` + "\r\n"
}

func (c typeCmd) execute(ctx context.Context, r *redisCommandProcessor, params commandParams) commandResult {
	if len(params) != 2 {
		return fmt.Sprintf("-ERR TYPE requires exactly one argument! %s", c.getUsage(ctx))
	}

	if _, exists := r.cache.Get(params[1]); exists {
		return "+string\r\n"
	}

	return "+none\r\n"
}
