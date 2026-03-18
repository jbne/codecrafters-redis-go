package redisserverlib

import (
	"context"
	"fmt"
	"strconv"

	redistypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/redis"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	lpop struct {
		*redisDataStore
	}
)

func (c lpop) getUsage(ctx context.Context) string {
	return `
usage:
	lpop key [count]
summary:
	Removes and returns the first elements of the list stored at key.
	By default, the command pops a single element from the beginning of the list.
	When provided with the optional count argument, the reply will consist of up to count elements, depending on the list's length.
`
}

func (c lpop) execute(ctx context.Context, params commandParams) commandResult {
	paramLen := len(params)
	if paramLen < 2 || paramLen > 3 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR LPOP requires 2 or 3 arguments!")}
	}

	listName := params[1].Val
	count := 1
	if paramLen == 3 {
		countStr := params[2].Val
		var err error
		count, err = strconv.Atoi(countStr)
		if err != nil {
			return resptypes.SimpleError{Val: fmt.Errorf("ERR Could not convert '%s' to an int for count! Err: %w", countStr, err)}
		}
	}

	if count < 1 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR Count must be a positive integer!")}
	}

	if entry, exists := c.dataStore.Get(listName); exists {
		if list, ok := entry.(redistypes.List); ok {
			result := resptypes.Array[resptypes.BulkString](list.PopFront(count))
			if count == 1 {
				return result[0]
			}

			return result
		}

		return resptypes.SimpleError{Val: fmt.Errorf("ERR LPOP can only be called on lists! %s", c.getUsage(ctx))}
	}

	return resptypes.NullBulkString
}
