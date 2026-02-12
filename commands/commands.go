package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/lib"
	"github.com/codecrafters-io/redis-starter-go/logger"
)

type (
	RESP2_Array []string

	RESP2_CommandRequest struct {
		Ctx    context.Context
		Params RESP2_Array
	}
	RESP2_CommandResponse = string
	RESP2_CommandHandlerSignature func(RESP2_CommandRequest) RESP2_CommandResponse
	RESP2_CommandsEntry struct {
		Execute RESP2_CommandHandlerSignature
	}
)

var (
	RESP2_Commands_Map = map[string]RESP2_CommandsEntry{
		"PING":  {Execute: PING},
		"ECHO":  {Execute: ECHO},

		// Cache commands
		"SET":   {Execute: SET},
		"GET":   {Execute: GET},

		// List commands
		"RPUSH": {Execute: RPUSH},
		"LRANGE": {Execute: LRANGE},
		"LPUSH": {Execute: LPUSH},
		"LLEN": {Execute: LLEN},
		"LPOP": {Execute: LPOP},
		"BLPOP": {Execute: BLPOP},
	}

	cache = lib.NewBlockingMap[string, string]()
	lists = lib.NewBlockingMap[string, []string]()
)

func BLPOP(request RESP2_CommandRequest) RESP2_CommandResponse {
	if len(request.Params) < 3 {
		return "-ERR BLPOP requires at least 2 arguments (list name and timeout)!\r\n"
	}

	timeoutSeconds, err := strconv.Atoi(request.Params[len(request.Params)-1])
	if err != nil {
		return fmt.Sprintf("-ERR Could not convert %s to an int for timeout! Err: %s\r\n", request.Params[len(request.Params)-1], err)
	}
	if timeoutSeconds < 0 {
		return "-ERR Timeout must be a non-negative integer!\r\n"
	}

	listName := request.Params[1]
	if _, exists := lists.Get(listName); exists {
		return LPOP(RESP2_CommandRequest{
			Ctx: request.Ctx,
			Params: []string{"LPOP", listName},
		})
	}

	timer := time.NewTimer(time.Duration(timeoutSeconds)*time.Second)
	if timeoutSeconds == 0 {
		// If timeout is 0, we should block indefinitely until an element is available. We can achieve this by not starting the timer at all and just waiting on a channel that will never receive anything.
		timer.Stop()
	}

	added := make(chan bool)
	onSet := make(chan bool)
	lists.OnSet = func(key string, value []string) {
		logger.Debug("OnSet called for key", "key", key)
		onSet <- true
	}
	
	go func() {
		select {
		case <-timer.C:
			added <- false
		case <-onSet:
			added <- true
		}
	}()

	if <-added {
		logger.Debug("BLPOP calling LPOP to remove element from list", "listName", listName)
		return LPOP(RESP2_CommandRequest{
			Ctx: request.Ctx,
			Params: []string{"LPOP", listName},
		})
	}

	return "$-1\r\n"
}

func LPOP(request RESP2_CommandRequest) RESP2_CommandResponse {
	if len(request.Params) < 2 {
		return "-ERR LPOP requires 2 or more arguments!\r\n"
	}
	
	listName := request.Params[1]
	list, exists := lists.Get(listName)
	if !exists || len(list) == 0 {
		return "$-1\r\n"
	}

	if len(request.Params) != 3 {
		value := list[0]
		lists.Set(listName, list[1:], 0)
		
		return fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)
	}

	count, err := strconv.Atoi(request.Params[2])
	if err != nil {
		return fmt.Sprintf("-ERR Could not convert %s to an int for count! Err: %s\r\n", request.Params[2], err)
	}
	if count < 1 {
		return "-ERR Count must be a positive integer!\r\n"
	}

	count = min(count, len(list))
	values := list[:count]
	lists.Set(listName, list[count:], 0)
	return lib.RespifyArray(values)
}

func LLEN(request RESP2_CommandRequest) RESP2_CommandResponse {
	if len(request.Params) != 2 {
		return "-ERR LLEN requires exactly 2 arguments (list name)!\r\n"
	}
	
	listName := request.Params[1]
	list, exists := lists.Get(listName)
	if !exists {
		return "-ERR List does not exist!\r\n"
	}
	return fmt.Sprintf(":%d\r\n", len(list))
}

func LPUSH(request RESP2_CommandRequest) RESP2_CommandResponse {
	if len(request.Params) < 3 {
		return "-ERR LPUSH requires at least 2 arguments (list name and value)!\r\n"
	}

	listName := request.Params[1]
	list, exists := lists.Get(listName)
	if !exists {
			list = []string{}
		}
	for _, v := range request.Params[2:] {
		list = append([]string{v}, list...)
	}
	lists.Set(listName, list, 0)
	
	return fmt.Sprintf(":%d\r\n", len(list))
}

func RPUSH(request RESP2_CommandRequest) RESP2_CommandResponse {
	if len(request.Params) < 3 {
		return "-ERR RPUSH requires at least 2 arguments (list name and value)!\r\n"
	}

	listName := request.Params[1]
	list, exists := lists.Get(listName)
	if !exists {
		list = []string{}
	}
	for _, v := range request.Params[2:] {
		list = append(list, v)
	}
	lists.Set(listName, list, 0)
	
	return fmt.Sprintf(":%d\r\n", len(list))
}

func LRANGE(request RESP2_CommandRequest) RESP2_CommandResponse {
	if len(request.Params) < 4 {
		return "-ERR LRANGE requires at least 3 arguments (list name, start index, end index)!\r\n"
	}

	listName := request.Params[1]
	if _, exists := lists.Get(listName); !exists {
		return "*0\r\n"
	}

	startIndex, err := strconv.Atoi(request.Params[2])
	if err != nil {
		return fmt.Sprintf("-ERR Could not convert %s to an int for start index! Err: %s\r\n", request.Params[2], err)
	}

	list, exists := lists.Get(listName)
	if !exists {
		return "*0\r\n"
	}
	if startIndex >= len(list) {
		return "*0\r\n"
	}

	stopIndex, err := strconv.Atoi(request.Params[3])
	if err != nil {
		return fmt.Sprintf("-ERR Could not convert %s to an int for end index! Err: %s\r\n", request.Params[3], err)
	}

	if startIndex < 0 {
		startIndex = max(len(list) + startIndex, 0) // Negative indices count from the end of the list, but we also need to ensure we don't go below 0
	}

	if stopIndex < 0 {
		stopIndex = len(list) + stopIndex
	}

	if startIndex > stopIndex {
		return "*0\r\n"
	}

	stopIndex = min(len(list), stopIndex + 1) // +1 because Redis LRANGE is inclusive, but Go slices are exclusive on the end index
	return lib.RespifyArray(list[startIndex:stopIndex])
}

func PING(request RESP2_CommandRequest) RESP2_CommandResponse {
	return "+PONG\r\n"
}

func ECHO(request RESP2_CommandRequest) RESP2_CommandResponse {
	tokens := request.Params
	if len(tokens) < 2 {
		return "-ERR No message provided to ECHO!\r\n"
	}
	if len(tokens) > 2 {
		return "-ERR ECHO accepts exactly 1 argument!\r\n"
	}
	response := tokens[1]
	return fmt.Sprintf("$%d\r\n%s\r\n", len(response), response)
}

func SET(request RESP2_CommandRequest) RESP2_CommandResponse {
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

func GET(request RESP2_CommandRequest) RESP2_CommandResponse {
	ctx := request.Ctx
	tokens := request.Params
	if len(tokens) < 2 {
		return "-ERR No key provided to GET!\r\n"
	}
	key := tokens[1]
	response, ok := cache.Get(key)

	if ok {
		logger.DebugContext(ctx, "GET cache hit", "key", key)
		return fmt.Sprintf("$%d\r\n%s\r\n", len(response), response)
	} else {
		logger.DebugContext(ctx, "GET cache miss", "key", key)
		return "$-1\r\n"
	}
}
