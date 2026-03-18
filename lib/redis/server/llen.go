package redisserverlib

import (
	"context"
	"fmt"

	redistypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/redis"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	llen struct {
		*redisDataStore
	}
)

func (c llen) getUsage(ctx context.Context) string {
	return `
usage:
	llen key
summary:
	Returns the length of the list stored at key.
	If key does not exist, it is interpreted as an empty list and 0 is returned.
	An error is returned when the value stored at key is not a list.
`
}

func (c llen) execute(ctx context.Context, params commandParams) commandResult {
	if len(params) != 2 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR LLEN requires key! %s", c.getUsage(ctx))}
	}

	listName := params[1].Val
	entry, exists := c.dataStore.Get(listName)
	if exists {
		if list, ok := entry.(redistypes.List); ok {
			return resptypes.Integer{Val: int64(list.Len())}
		}

		return resptypes.SimpleError{Val: fmt.Errorf("ERR LLEN can only be called on lists! %s", c.getUsage(ctx))}
	}

	return resptypes.Integer{Val: 0}
}
