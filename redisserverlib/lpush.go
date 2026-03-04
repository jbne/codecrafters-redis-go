package redisserverlib

import (
	"context"
	"fmt"
)

type (
	lpush struct{}
)

func (c lpush) getUsage(ctx context.Context) string {
	return `
usage:
	lpush key element [element ...]
summary:
	Insert all the specified values at the head of the list stored at key.
	If key does not exist, it is created as empty list before performing the push operations.
	When key holds a value that is not a list, an error is returned.
` + "\r\n"
}

func (c lpush) execute(ctx context.Context, r *redisCommandProcessor, params commandParams) commandResult {
	if len(params) < 3 {
		return fmt.Sprintf("-ERR LPUSH requires key and at least one element! %s", c.getUsage(ctx))
	}

	listName := params[1]
	entry := r.dataStore.GetOrCreate(listName, newRedisListAny, 0)
	if list, ok := entry.(redisType_List); ok {
		newLen := list.PushFront(params[2:]...)
		return fmt.Sprintf(":%d\r\n", newLen)
	}

	return fmt.Sprintf("-ERR LPUSH can only be called on lists! %s", c.getUsage(ctx))
}
