package redisserverlib

import (
	"context"
	"fmt"

	redistypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/redis"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	llen struct {
		redistypes.DataStore
	}
)

func (c llen) moniker() string {
	return "LLEN"
}

func (c llen) getUsage() string {
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
		return resptypes.SimpleError{Val: fmt.Errorf("ERR LLEN requires key! %s", c.getUsage())}
	}

	listName := params[1].Val
	dsVal, exists := c.Get(listName)
	if exists {
		if dsVal.Type != redistypes.TypeList {
			return resptypes.SimpleError{Val: fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")}
		}

		return resptypes.Integer{Val: int64(dsVal.List.Len())}
	}

	return resptypes.Integer{Val: 0}
}
