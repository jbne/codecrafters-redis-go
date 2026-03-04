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

	if entry, exists := r.dataStore.Get(params[1]); exists {
		switch entry.(type) {
		case redisType_String:
			return "+string\r\n"
		case redisType_List:
			return "+list\r\n"
		case redisType_Stream:
			return "+stream\r\n"
		default:
			return "+unknown\r\n"
		}
	}

	return "+none\r\n"
}
