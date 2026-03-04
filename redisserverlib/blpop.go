package redisserverlib

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/redislib"
)

type (
	blpop struct{}
)

func (c blpop) getUsage(ctx context.Context) string {
	return `
usage:
	blpop key [key ...] timeout
summary:
	BLPOP is a blocking list pop primitive.
	It is the blocking version of LPOP because it blocks the connection when there are no elements to pop from any of the given lists.
	An element is popped from the head of the first list that is non-empty, with the given keys being checked in the order that they are given.

	BLPOP currently only supports popping from one list.
` + "\r\n"
}

func (c blpop) execute(ctx context.Context, r *redisCommandProcessor, params commandParams) commandResult {
	paramLen := len(params)
	if paramLen != 3 {
		return "-ERR BLPOP requires 3 arguments!\r\n"
	}

	timeoutSeconds, err := strconv.ParseFloat(params[2], 32)
	if err != nil {
		return fmt.Sprintf("-ERR Could not convert '%s' to a float for timeoutSeconds! Err: %s\r\n", params[2], err)
	}
	if timeoutSeconds < 0 {
		return "-ERR TimeoutSeconds must be a non-negative integer!\r\n"
	}

	var cancel context.CancelFunc
	if timeoutSeconds > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds*1000)*time.Millisecond)
		defer cancel()
	}

	listName := params[1]
	entry := r.dataStore.GetOrCreate(listName, newRedisListAny, 0)
	if list, ok := entry.(redisType_List); ok {
		err, result := list.PopFrontAsync(ctx)

		if err != nil {
			slog.DebugContext(ctx, "BLPOP error occurred", "listName", listName, "error", err)
			if err == context.DeadlineExceeded {
				return "*-1\r\n"
			}
			return ""
		}

		result = append([]string{listName}, result...)
		return redislib.SerializeRespArray(result)
	}

	return fmt.Sprintf("-ERR BLPOP can only be called on lists! %s", c.getUsage(ctx))
}
