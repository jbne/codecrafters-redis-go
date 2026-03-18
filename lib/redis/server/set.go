package redisserverlib

import (
	"context"
	"fmt"
	"strconv"
	"time"

	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	set struct {
		*redisDataStore
	}
)

func (c set) getUsage(ctx context.Context) string {
	return `
usage:
	set key value [PX milliseconds]

summary:
	Set key to hold the string value.
	If key already holds a value, it is overwritten, regardless of its type.
	Any previous time to live associated with the key is discarded on successful SET operation.
` + "\r\n"
}

func (c set) execute(ctx context.Context, params commandParams) commandResult {
	tokens := params
	arrSize := len(tokens)
	switch {
	case arrSize >= 3:
		key := tokens[1].Val
		value := tokens[2].Val

		// Validate key and value are not empty
		if key == "" {
			return resptypes.SimpleError{Val: fmt.Errorf("ERR Key cannot be empty!")}
		}
		if value == "" {
			return resptypes.SimpleError{Val: fmt.Errorf("ERR Value cannot be empty!")}
		}

		expiryDurationMs := 0
		err := error(nil)
		for i := 3; i < arrSize; i++ {
			if tokens[i].Val == "PX" {
				if i+1 >= arrSize {
					return resptypes.SimpleError{Val: fmt.Errorf("ERR No expiration specified!")}
				}

				expiry := tokens[i+1].Val
				expiryDurationMs, err = strconv.Atoi(expiry)
				if err != nil {
					return resptypes.SimpleError{Val: fmt.Errorf("ERR Could not convert %s to an int for expiry! Err: %w", expiry, err)}
				}

				break
			}
		}

		entry := c.dataStore.GetOrCreate(key, func() any { return tokens[2] }, time.Duration(expiryDurationMs)*time.Millisecond)
		if _, ok := entry.(resptypes.BulkString); !ok {
			return resptypes.SimpleError{Val: fmt.Errorf("ERR SET command can only be called on string values! %s", c.getUsage(ctx))}
		}

		return resptypes.SimpleString{Val: "OK"}
	case arrSize == 2:
		return resptypes.SimpleError{Val: fmt.Errorf("ERR No value given for key %s!", tokens[1].Val)}
	case arrSize == 1:
		return resptypes.SimpleError{Val: fmt.Errorf("ERR No key given!")}
	default:
		return resptypes.SimpleError{Val: fmt.Errorf("ERR SET command accepts at most 2 arguments (key and value) plus optional PX expiry!")}
	}
}
