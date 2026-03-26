package redisserverlib

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	redistypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/redis"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	blpop struct {
		redistypes.DataStore
	}
)

func (c blpop) moniker() string {
	return "BLPOP"
}

func (c blpop) getUsage() string {
	return `
usage:
	blpop key timeout
summary:
	BLPOP is a blocking list pop primitive.
	It is the blocking version of LPOP because it blocks the connection when there are no elements to pop from any of the given lists.
	An element is popped from the head of the first list that is non-empty, with the given keys being checked in the order that they are given.

	BLPOP currently only supports popping from one list.
`
}

func (c blpop) execute(ctx context.Context, params commandParams) commandResult {
	paramLen := len(params)
	if paramLen != 3 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR BLPOP requires 3 arguments!")}
	}

	listName := params[1].Val
	timeoutStr := params[2].Val

	timeoutSeconds, err := strconv.ParseFloat(timeoutStr, 32)
	if err != nil {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR Could not convert '%s' to a float for timeoutSeconds! Err: %w", timeoutStr, err)}
	}
	if timeoutSeconds < 0 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR TimeoutSeconds must be a non-negative integer!")}
	}

	var cancel context.CancelFunc
	if timeoutSeconds > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds*1000)*time.Millisecond)
		defer cancel()
	}

	dsVal := c.GetOrCreate(listName, redistypes.NewList, 0)
	if dsVal.Type != redistypes.TypeList {
		return resptypes.SimpleError{Val: fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")}
	}

	err, result := dsVal.List.PopFrontAsync(ctx)

	if err != nil {
		slog.DebugContext(ctx, "BLPOP error occurred", "listName", listName, "error", err)
		if err == context.DeadlineExceeded {
			return resptypes.NullArray
		}

		return resptypes.Null{}
	}

	result = append([]resptypes.BulkString{params[1]}, result...)
	return resptypes.Array[resptypes.BulkString](result)
}
