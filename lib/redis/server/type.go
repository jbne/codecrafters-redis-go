package redisserverlib

import (
	"context"
	"fmt"

	redistypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/redis"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	typeCmd struct {
		*redisDataStore
	}
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

func (c typeCmd) execute(ctx context.Context, params commandParams) commandResult {
	if len(params) != 2 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR TYPE requires exactly one argument! %s", c.getUsage(ctx))}
	}

	key := params[1].Val
	typeString := "none"
	if entry, exists := c.dataStore.Get(key); exists {
		switch entry.(type) {
		case resptypes.BulkString:
			typeString = "string"
		case redistypes.List:
			typeString = "list"
		case redistypes.Stream:
			typeString = "stream"
		default:
			typeString = "unknown"
		}
	}

	return resptypes.SimpleString{Val: typeString}
}
