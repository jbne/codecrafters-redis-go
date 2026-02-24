package respcommands

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/concurrent"
	"github.com/codecrafters-io/redis-starter-go/resplib"
)

type (
	set struct{}
	get struct{}
)

var (
	cache = concurrent.NewConcurrentMap[string, string]()
)

func (c set) getUsage(ctx context.Context) string {
	return `
usage:
	set key value [NX | XX | IFEQ ifeq-value | IFNE ifne-value | IFDEQ ifdeq-digest | IFDNE ifdne-digest] [get] [EX seconds | PX milliseconds | EXAT unix-time-seconds | PXAT unix-time-milliseconds | KEEPTTL]

summary:
	Set key to hold the string value. If key already holds a value, it is overwritten, regardless of its type. Any previous time to live associated with the key is discarded on successful SET operation.
	Only PX milliseconds expiry option is implemented for simplicity - other options are ignored.
` + "\r\n"
}

func (c set) execute(ctx context.Context, request resplib.RESP2_CommandRequest) resplib.RESP2_CommandResponse {
	tokens := request.Params
	arrSize := len(tokens)
	switch {
	case arrSize >= 3:
		key := tokens[1]
		value := tokens[2]

		// Validate key and value are not empty
		if key == "" {
			return "-ERR Key cannot be empty!\r\n"
		}
		if value == "" {
			return "-ERR Value cannot be empty!\r\n"
		}

		expiryDurationMs := 0
		err := error(nil)
		for i := 3; i < arrSize; i++ {
			if tokens[i] == "PX" {
				if i+1 >= arrSize {
					return "-ERR No expiration specified!\r\n"
				} else {
					expiryDurationMs, err = strconv.Atoi(tokens[i+1])
					if err != nil {
						return fmt.Sprintf("-ERR Could not convert %s to an int for expiry! Err: %s\r\n", tokens[i+1], err)
					}
				}
			}
		}

		cache.Set(key, value, time.Duration(expiryDurationMs)*time.Millisecond)

		return "+OK\r\n"
	case arrSize == 2:
		return fmt.Sprintf("-ERR No value given for key %s!\r\n", tokens[1])
	case arrSize == 1:
		return "-ERR No key given!\r\n"
	default:
		return "-ERR SET command accepts at most 2 arguments (key and value) plus optional PX expiry!\r\n"
	}
}

func (c get) getUsage(ctx context.Context) string {
	return `
usage:
	get key
summary:
	Get the value of key. If the key does not exist the special value nil is returned.
	An error is returned if the value stored at key is not a string, because GET only handles string values.
` + "\r\n"
}

func (c get) execute(ctx context.Context, request resplib.RESP2_CommandRequest) resplib.RESP2_CommandResponse {
	if len(request.Params) < 2 {
		return "-ERR No key provided to GET!\r\n"
	}
	key := request.Params[1]
	response, ok := cache.Get(key)

	if ok {
		slog.DebugContext(ctx, "GET cache hit", "key", key)
		return fmt.Sprintf("$%d\r\n%s\r\n", len(response), response)
	}

	slog.DebugContext(ctx, "GET cache miss", "key", key)
	return "$-1\r\n"
}
