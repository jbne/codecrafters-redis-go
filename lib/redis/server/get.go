package redisserverlib

import (
	"context"
	"fmt"

	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	get struct {
		*redisDataStore
	}
)

func (c get) getUsage(ctx context.Context) string {
	return `
usage:
	get key
summary:
	Get the value of key. If the key does not exist the special value nil is returned.
	An error is returned if the value stored at key is not a string, because GET only handles string values.
`
}

func (c get) execute(ctx context.Context, params commandParams) commandResult {
	if len(params) < 2 {
		return resptypes.Error{Val: fmt.Errorf("ERR No key provided to GET!")}
	}

	key := params[1].Val
	if entry, exists := c.dataStore.Get(key); exists {
		if value, ok := entry.(resptypes.BulkString); ok {
			return value
		}

		return resptypes.Error{Val: fmt.Errorf("ERR GET can only be called on string values! %s", c.getUsage(ctx))}
	}

	return resptypes.BulkString{Length: -1}
}
