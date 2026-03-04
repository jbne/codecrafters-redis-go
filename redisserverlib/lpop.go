package redisserverlib

import (
	"context"
	"fmt"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/redislib"
)

type (
	lpop struct{}
)

func (c lpop) getUsage(ctx context.Context) string {
	return `
usage:
	lpop key [count]
summary:
	Removes and returns the first elements of the list stored at key.
	By default, the command pops a single element from the beginning of the list.
	When provided with the optional count argument, the reply will consist of up to count elements, depending on the list's length.
` + "\r\n"
}

func (c lpop) execute(ctx context.Context, r *redisCommandProcessor, params commandParams) commandResult {
	paramLen := len(params)
	if paramLen < 2 || paramLen > 3 {
		return "-ERR LPOP requires 2 or 3 arguments!\r\n"
	}

	count := 1
	if paramLen == 3 {
		var err error
		count, err = strconv.Atoi(params[2])
		if err != nil {
			return fmt.Sprintf("-ERR Could not convert '%s' to an int for count! Err: %s\r\n", params[2], err)
		}
	}

	if count < 1 {
		return "-ERR Count must be a positive integer!\r\n"
	}

	listName := params[1]
	if entry, exists := r.dataStore.Get(listName); exists {
		if list, ok := entry.(redisType_List); ok {
			result := list.PopFront(count)
			if count == 1 {
				return fmt.Sprintf("$%d\r\n%s\r\n", len(result[0]), result[0])
			}

			return redislib.SerializeRespArray(result)
		}

		return fmt.Sprintf("-ERR LPOP can only be called on lists! %s", c.getUsage(ctx))
	}

	return "$-1\r\n"
}
