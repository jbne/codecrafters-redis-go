package redisserverlib

import (
	"context"
	"fmt"
	"strconv"

	redistypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/redis"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	lrange struct {
		redistypes.DataStore
	}
)

func (c lrange) moniker() string {
	return "LRANGE"
}

func (c lrange) getUsage() string {
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
`
}

func (c lrange) execute(ctx context.Context, params commandParams) commandResult {
	if len(params) != 4 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR LRANGE key, start, and stop! %s", c.getUsage())}
	}

	listName := params[1].Val
	startIndexStr := params[2].Val
	stopIndexStr := params[3].Val

	startIndex, err := strconv.Atoi(startIndexStr)
	if err != nil {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR Start index '%s' could not be converted to int! Err: %w", startIndexStr, err)}
	}

	stopIndex, err := strconv.Atoi(stopIndexStr)
	if err != nil {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR Stop index '%s' could not be converted to int! Err: %w", stopIndexStr, err)}
	}

	if dsVal, exists := c.Get(listName); exists {
		if dsVal.Type != redistypes.TypeList {
			return resptypes.SimpleError{Val: fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")}
		}

		bulkStrings := resptypes.Array[resptypes.BulkString](dsVal.List.GetRange(startIndex, stopIndex))
		return bulkStrings
	}

	return resptypes.Array[resptypes.RespSerializable]{}
}
