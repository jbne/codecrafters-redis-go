package redisserverlib

import (
	"context"
	"fmt"
)

type (
	llen struct{}
)

func (c llen) getUsage(ctx context.Context) string {
	return `
usage:
	llen key
summary:
	Returns the length of the list stored at key.
	If key does not exist, it is interpreted as an empty list and 0 is returned.
	An error is returned when the value stored at key is not a list.
` + "\r\n"
}

func (c llen) execute(ctx context.Context, r *redisCommandProcessor, params commandParams) commandResult {
	if len(params) != 2 {
		return fmt.Sprintf("-ERR LLEN requires key! %s", c.getUsage(ctx))
	}

	listName := params[1]
	entry, exists := r.dataStore.Get(listName)
	if exists {
		if list, ok := entry.(redisType_List); ok {
			return fmt.Sprintf(":%d\r\n", list.Len())
		}

		return fmt.Sprintf("-ERR LLEN can only be called on lists! %s", c.getUsage(ctx))
	}

	return ":0\r\n"
}
