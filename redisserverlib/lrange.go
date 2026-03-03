package redisserverlib

import (
	"context"
	"fmt"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/redislib"
)

type (
	lrange struct{}
)

func (c lrange) getUsage(ctx context.Context) string {
	return `
usage:
	lrange key start stop
summary:
	Returns the specified elements of the list stored at key.
	The offsets start and stop are zero-based indexes, with 0 being the
	first element of the list (the head of the list), 1 being the next element and so on.

	These offsets can also be negative numbers indicating offsets starting at the end of the list.
	For example, -1 is the last element of the list, -2 the penultimate, and so on.

	Out of range indexes will not produce an error.
	If start is larger than the end of the list, an empty list is returned.
	If stop is larger than the actual end of the list, Redis will treat it like the last element of the list.
` + "\r\n"
}

func (c lrange) execute(ctx context.Context, r *redisCommandProcessor, params commandParams) commandResult {
	if len(params) != 4 {
		return fmt.Sprintf("-ERR LRANGE key, start, and stop! %s", c.getUsage(ctx))
	}

	startIndex, err := strconv.Atoi(params[2])
	if err != nil {
		return fmt.Sprintf("-ERR Start index '%s' could not be converted to int! Err: %s\r\n", params[2], err)
	}

	stopIndex, err := strconv.Atoi(params[3])
	if err != nil {
		return fmt.Sprintf("-ERR Stop index '%s' could not be converted to int! Err: %s\r\n", params[3], err)
	}

	listName := params[1]
	list, exists := r.lists.Get(listName)
	if !exists {
		return "*0\r\n"
	}

	return redislib.SerializeRespArray(list.GetRange(startIndex, stopIndex))
}
