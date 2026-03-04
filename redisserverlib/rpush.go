package redisserverlib

import (
	"context"
	"fmt"
)

type (
	rpush struct{}
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

func (c rpush) execute(ctx context.Context, r *redisCommandProcessor, params commandParams) commandResult {
	if len(params) < 3 {
		return fmt.Sprintf("-ERR RPUSH requires key and at least one element! %s", c.getUsage(ctx))
	}

	listName := params[1]
	entry := r.dataStore.GetOrCreate(listName, newRedisListAny, 0)

	if list, ok := entry.(redisType_List); ok {
		newLen := list.PushBack(params[2:]...)
		return fmt.Sprintf(":%d\r\n", newLen)
	}

	return fmt.Sprintf("-ERR RPUSH can only be called on lists! %s", c.getUsage(ctx))
}
