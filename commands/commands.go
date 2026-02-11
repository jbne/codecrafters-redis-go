package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/codecrafters-io/redis-starter-go/logger"
)

type (
	RESP2_Array []string

	RESP2_CommandHandlerParams struct {
		Ctx    context.Context
		Params RESP2_Array
	}
	RESP2_CommandHandlerReturn = string
	RESP2_CommandHandlerSignature func(args RESP2_CommandHandlerParams) RESP2_CommandHandlerReturn
	RESP2_CommandsMapEntry struct {
		Execute RESP2_CommandHandlerSignature
	}
)

var (
	RESP2_Commands_Map = map[string]RESP2_CommandsMapEntry{
		"PING":  {Execute: PING},
		"ECHO":  {Execute: ECHO},
		"SET":   {Execute: SET},
		"GET":   {Execute: GET},
		"RPUSH": {Execute: RPUSH},
		"LRANGE": {Execute: LRANGE},
	}

	cache       = map[string]string{}
	cacheMutex  sync.RWMutex
	timers      = map[string]*time.Timer{}
	timersMutex sync.Mutex

	lists = map[string][]string{}
)

func RPUSH(params RESP2_CommandHandlerParams) RESP2_CommandHandlerReturn {
	if len(params.Params) < 3 {
		return "-ERR RPUSH requires at least 2 arguments (list name and value)!\r\n"
	}

	listName := params.Params[1]
	for _, v := range params.Params[2:] {
		lists[listName] = append(lists[listName], v)
	}
	
	return fmt.Sprintf(":%d\r\n", len(lists[listName]))
}

func LRANGE(params RESP2_CommandHandlerParams) RESP2_CommandHandlerReturn {
	if len(params.Params) < 4 {
		return "-ERR LRANGE requires at least 3 arguments (list name, start index, end index)!\r\n"
	}

	listName := params.Params[1]
	if _, exists := lists[listName]; !exists {
		return "*0\r\n"
	}

	startIndex, err := strconv.Atoi(params.Params[2])
	if err != nil {
		return fmt.Sprintf("-ERR Could not convert %s to an int for start index! Err: %s\r\n", params.Params[2], err)
	}

	if startIndex >= len(lists[listName]) {
		return "*0\r\n"
	}

	stopIndex, err := strconv.Atoi(params.Params[3])
	if err != nil {
		return fmt.Sprintf("-ERR Could not convert %s to an int for end index! Err: %s\r\n", params.Params[3], err)
	}

	if startIndex > stopIndex {
		return "*0\r\n"
	}

	stopIndex = min(len(lists[listName]), stopIndex)
	return RespifyArray(lists[listName][startIndex:stopIndex])	
}

func RespifyArray(s []string) RESP2_CommandHandlerReturn {
	for i, str := range s {
		s[i] = fmt.Sprintf("$%d\r\n%s\r\n", len(str), str)
	}
	return fmt.Sprintf("*%d\r\n%s", len(s), strings.Join(s, ""))
}

func PING(params RESP2_CommandHandlerParams) RESP2_CommandHandlerReturn {
	return "+PONG\r\n"
}

func ECHO(params RESP2_CommandHandlerParams) RESP2_CommandHandlerReturn {
	tokens := params.Params
	if len(tokens) < 2 {
		return "-ERR No message provided to ECHO!\r\n"
	}
	if len(tokens) > 2 {
		return "-ERR ECHO accepts exactly 1 argument!\r\n"
	}
	response := tokens[1]
	return fmt.Sprintf("$%d\r\n%s\r\n", len(response), response)
}

func SET(args RESP2_CommandHandlerParams) RESP2_CommandHandlerReturn {
	tokens := args.Params
	ctx := args.Ctx
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

		// Stop any existing timer for this key
		timersMutex.Lock()
		if timer, exists := timers[key]; exists {
			logger.DebugContext(context.Background(), "Cancelling existing timer", "key", key)
			timer.Stop()
			delete(timers, key)
		}
		timersMutex.Unlock()

		cacheMutex.Lock()
		cache[key] = value
		cacheMutex.Unlock()
		logger.DebugContext(ctx, "SET executed", "key", key, "value", value, "expiry_ms", expiryDurationMs)

		if expiryDurationMs > 0 {
			timer := time.NewTimer(time.Millisecond * time.Duration(expiryDurationMs))
			timersMutex.Lock()
			timers[key] = timer
			timersMutex.Unlock()

			go func() {
				<-timer.C

				cacheMutex.Lock()
				delete(cache, key)
				cacheMutex.Unlock()

				timersMutex.Lock()
				delete(timers, key)
				timersMutex.Unlock()
				logger.DebugContext(ctx, "Key expired", "key", key)
			}()
		}

		return "+OK\r\n"
	case arrSize == 2:
		return fmt.Sprintf("-ERR No value given for key %s!\r\n", tokens[1])
	case arrSize == 1:
		return "-ERR No key given!\r\n"
	default:
		return "-ERR SET command accepts at most 2 arguments (key and value) plus optional PX expiry!\r\n"
	}
}

func GET(args RESP2_CommandHandlerParams) RESP2_CommandHandlerReturn {
	ctx := args.Ctx
	tokens := args.Params
	if len(tokens) < 2 {
		return "-ERR No key provided to GET!\r\n"
	}
	key := tokens[1]
	cacheMutex.RLock()
	response, ok := cache[key]
	cacheMutex.RUnlock()

	if ok {
		logger.DebugContext(ctx, "GET cache hit", "key", key)
		return fmt.Sprintf("$%d\r\n%s\r\n", len(response), response)
	} else {
		logger.DebugContext(ctx, "GET cache miss", "key", key)
		return "$-1\r\n"
	}
}
