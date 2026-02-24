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

func (c set) execute(ctx context.Context, request resplib.RESP2_CommandRequest) {
	tokens := request.Params
	arrSize := len(tokens)
	switch {
	case arrSize >= 3:
		key := tokens[1]
		value := tokens[2]

		// Validate key and value are not empty
		if key == "" {
			request.ResponseChannel <- "-ERR Key cannot be empty!\r\n"
			return
		}
		if value == "" {
			request.ResponseChannel <- "-ERR Value cannot be empty!\r\n"
			return
		}

		expiryDurationMs := 0
		err := error(nil)
		for i := 3; i < arrSize; i++ {
			if tokens[i] == "PX" {
				if i+1 >= arrSize {
					request.ResponseChannel <- "-ERR No expiration specified!\r\n"
					return
				}

				expiryDurationMs, err = strconv.Atoi(tokens[i+1])
				if err != nil {
					request.ResponseChannel <- fmt.Sprintf("-ERR Could not convert %s to an int for expiry! Err: %s\r\n", tokens[i+1], err)
					return
				}
			}
		}

		cache.Set(key, value, time.Duration(expiryDurationMs)*time.Millisecond)

		request.ResponseChannel <- "+OK\r\n"
	case arrSize == 2:
		request.ResponseChannel <- fmt.Sprintf("-ERR No value given for key %s!\r\n", tokens[1])
	case arrSize == 1:
		request.ResponseChannel <- "-ERR No key given!\r\n"
	default:
		request.ResponseChannel <- "-ERR SET command accepts at most 2 arguments (key and value) plus optional PX expiry!\r\n"
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

func (c get) execute(ctx context.Context, request resplib.RESP2_CommandRequest) {
	if len(request.Params) < 2 {
		request.ResponseChannel <- "-ERR No key provided to GET!\r\n"
		return
	}
	key := request.Params[1]
	response, ok := cache.Get(key)

	if ok {
		slog.DebugContext(ctx, "GET cache hit", "key", key)
		request.ResponseChannel <- fmt.Sprintf("$%d\r\n%s\r\n", len(response), response)
		return
	}

	slog.DebugContext(ctx, "GET cache miss", "key", key)
	request.ResponseChannel <- "$-1\r\n"
}
