package redisserverlib

import (
	"context"
	"fmt"

	redistypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/redis"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	rpush struct {
		*redisDataStore
	}
)

func (c rpush) getUsage(ctx context.Context) string {
	return `
usage:
	rpush key element [element ...]
summary:
	Insert all the specified values at the tail of the list stored at key.
	If key does not exist, it is created as empty list before performing the push operation.
	When key holds a value that is not a list, an error is returned.
` + "\r\n"
}

func (c rpush) execute(ctx context.Context, params commandParams) commandResult {
	if len(params) < 3 {
		return resptypes.Error{Val: fmt.Errorf("ERR RPUSH requires key and at least one element! %s", c.getUsage(ctx))}
	}

	listName := params[1].Val
	entry := c.dataStore.GetOrCreate(listName, redistypes.NewList, 0)

	if list, ok := entry.(redistypes.List); ok {
		newLen := list.PushBack(params[2:]...)
		return resptypes.Integer{Val: int64(newLen)}
	}

	return resptypes.Error{Val: fmt.Errorf("ERR RPUSH can only be called on lists! %s", c.getUsage(ctx))}
}
