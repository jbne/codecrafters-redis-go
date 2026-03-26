package redisserverlib

import (
	"context"
	"fmt"

	redistypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/redis"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	lpush struct {
		redistypes.DataStore
	}
)

func (c lpush) moniker() string {
	return "LPUSH"
}

func (c lpush) getUsage() string {
	return `
usage:
	lpush key element [element ...]
summary:
	Insert all the specified values at the head of the list stored at key.
	If key does not exist, it is created as empty list before performing the push operations.
	When key holds a value that is not a list, an error is returned.
`
}

func (c lpush) execute(ctx context.Context, params commandParams) commandResult {
	if len(params) < 3 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR LPUSH requires key and at least one element! %s", c.getUsage())}
	}

	listName := params[1].Val
	dsVal := c.GetOrCreate(listName, redistypes.NewList, 0)
	if dsVal.Type != redistypes.TypeList {
		return resptypes.SimpleError{Val: fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")}
	}

	newLen := dsVal.List.PushFront(params[2:]...)
	return resptypes.Integer{Val: int64(newLen)}
}
