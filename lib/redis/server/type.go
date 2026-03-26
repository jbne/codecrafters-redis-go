package redisserverlib

import (
	"context"
	"fmt"

	redistypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/redis"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	typeCmd struct {
		redistypes.DataStore
	}
)

func (c typeCmd) moniker() string {
	return "TYPE"
}

func (c typeCmd) getUsage() string {
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
		return resptypes.SimpleError{Val: fmt.Errorf("ERR TYPE requires exactly one argument! %s", c.getUsage())}
	}

	key := params[1].Val
	typeString := "none"
	if dsVal, exists := c.Get(key); exists {
		switch dsVal.Type {
		case redistypes.TypeString:
			typeString = "string"
		case redistypes.TypeList:
			typeString = "list"
		case redistypes.TypeStream:
			typeString = "stream"
		case redistypes.TypeUnknown:
			typeString = "unknown"
		}
	}

	return resptypes.SimpleString{Val: typeString}
}
