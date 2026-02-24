package respcommands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/concurrent"
	"github.com/codecrafters-io/redis-starter-go/resplib"
)

var (
	lists = concurrent.NewConcurrentMap[string, *concurrent.ConcurrentDeque[string]]()
)

type (
	rpush  struct{}
	lrange struct{}
	lpush  struct{}
	llen   struct{}
	lpop   struct{}
	blpop  struct{}
)

func (c rpush) getUsage(ctx context.Context) string {
	return `
usage:
	rpush key element [element ...]
summary:
	Insert all the specified values at the tail of the list stored at key.
	If key does not exist, it is created as empty list before performing the push operation.
	When key holds a value that is not a list, an error is returned.
` + "\r\n"
}

func (c rpush) execute(ctx context.Context, request resplib.RESP2_CommandRequest) resplib.RESP2_CommandResponse {
	if len(request.Params) < 3 {
		return fmt.Sprintf("-ERR RPUSH requires key and at least one element! %s", c.getUsage(ctx))
	}

	listName := request.Params[1]
	list := lists.GetOrCreate(listName, concurrent.NewConcurrentDeque[string])
	newLen := list.PushBack(request.Params[2:]...)
	return fmt.Sprintf(":%d\r\n", newLen)
}

func (c lrange) getUsage(ctx context.Context) string {
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
` + "\r\n"
}

func (c lrange) execute(ctx context.Context, request resplib.RESP2_CommandRequest) resplib.RESP2_CommandResponse {
	if len(request.Params) != 4 {
		return fmt.Sprintf("-ERR LRANGE key, start, and stop! %s", c.getUsage(ctx))
	}

	startIndex, err := strconv.Atoi(request.Params[2])
	if err != nil {
		return fmt.Sprintf("-ERR Start index '%s' could not be converted to int! Err: %s\r\n", request.Params[2], err)
	}

	stopIndex, err := strconv.Atoi(request.Params[3])
	if err != nil {
		return fmt.Sprintf("-ERR Stop index '%s' could not be converted to int! Err: %s\r\n", request.Params[3], err)
	}

	listName := request.Params[1]
	list, exists := lists.Get(listName)
	if !exists {
		return "*0\r\n"
	}

	return resplib.SerializeRespArray(list.GetRange(startIndex, stopIndex))
}

func (c lpush) getUsage(ctx context.Context) string {
	return `
usage:
	lpush key element [element ...]
summary:
	Insert all the specified values at the head of the list stored at key.
	If key does not exist, it is created as empty list before performing the push operations.
	When key holds a value that is not a list, an error is returned.
` + "\r\n"
}

func (c lpush) execute(ctx context.Context, request resplib.RESP2_CommandRequest) resplib.RESP2_CommandResponse {
	if len(request.Params) < 3 {
		return fmt.Sprintf("-ERR LPUSH requires key and at least one element! %s", c.getUsage(ctx))
	}

	listName := request.Params[1]
	list := lists.GetOrCreate(listName, concurrent.NewConcurrentDeque[string])
	newLen := list.PushFront(request.Params[2:]...)
	return fmt.Sprintf(":%d\r\n", newLen)
}

func (c llen) getUsage(ctx context.Context) string {
	return `
usage:
	llen key
summary:
	Returns the length of the list stored at key.
	If key does not exist, it is interpreted as an empty list and 0 is returned.
	An error is returned when the value stored at key is not a list.
` + "\r\n"
}

func (c llen) execute(ctx context.Context, request resplib.RESP2_CommandRequest) resplib.RESP2_CommandResponse {
	if len(request.Params) != 2 {
		return fmt.Sprintf("-ERR LLEN requires key! %s", c.getUsage(ctx))
	}

	listName := request.Params[1]
	list, exists := lists.Get(listName)
	if !exists {
		return "-ERR List does not exist!\r\n"
	}

	return fmt.Sprintf(":%d\r\n", list.Len())
}

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

func (c lpop) execute(ctx context.Context, request resplib.RESP2_CommandRequest) resplib.RESP2_CommandResponse {
	paramLen := len(request.Params)
	if paramLen < 2 || paramLen > 3 {
		return "-ERR LPOP requires 2 or 3 arguments!\r\n"
	}

	count := 1
	if paramLen == 3 {
		var err error
		count, err = strconv.Atoi(request.Params[2])
		if err != nil {
			return fmt.Sprintf("-ERR Could not convert '%s' to an int for count! Err: %s\r\n", request.Params[2], err)
		}
	}

	if count < 1 {
		return "-ERR Count must be a positive integer!\r\n"
	}

	listName := request.Params[1]
	if list, exists := lists.Get(listName); exists {
		return resplib.SerializeRespArray(list.PopFront(count))
	}

	return "$-1\r\n"
}

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

func (c blpop) execute(ctx context.Context, request resplib.RESP2_CommandRequest) resplib.RESP2_CommandResponse {
	paramLen := len(request.Params)
	if paramLen != 3 {
		return "-ERR BLPOP requires 3 arguments!\r\n"
	}

	timeoutSeconds, err := strconv.Atoi(request.Params[2])
	if err != nil {
		return fmt.Sprintf("-ERR Could not convert '%s' to an int for timeoutSeconds! Err: %s\r\n", request.Params[2], err)
	}

	listName := request.Params[1]
	list := lists.GetOrCreate(listName, concurrent.NewConcurrentDeque[string])
	return resplib.SerializeRespArray(<-list.PopFrontAsync(time.Duration(timeoutSeconds) * time.Second))
}
