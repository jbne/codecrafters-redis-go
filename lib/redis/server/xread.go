package redisserverlib

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	redistypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/redis"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	xread struct {
		redistypes.DataStore
	}
)

func (c xread) moniker() string {
	return "XREAD"
}

func (c xread) getUsage() string {
	return `
usage:
	XREAD STREAMS key [key ...] id [id ...]
summary:
	Read data from one or multiple streams, only returning entries with an ID greater than the last received ID reported by the caller.
	This command has an option to block if items are not available, in a similar fashion to BRPOP or BZPOPMIN and others.

	Please note that before reading this page, if you are new to streams, we recommend to read our introduction to Redis Streams.
`
}

func (c xread) execute(ctx context.Context, params commandParams) commandResult {
	if len(params) != 4 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR XRANGE requires exactly 4 arguments! %s", c.getUsage())}
	}

	streams := params[1].Val
	if strings.ToUpper(streams) != "STREAMS" {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR Expected STREAMS option!")}
	}

	streamKey := params[2]
	streamKeyStr := streamKey.Val
	dsVal, exists := c.Get(streamKeyStr)
	if !exists {
		return resptypes.Array[resptypes.RespSerializable]{}
	}

	if dsVal.Type != redistypes.TypeStream {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR XRANGE can only be called on streams! %s", c.getUsage())}
	}

	startStr := params[3].Val
	start := redistypes.StreamEntryId{}
	if startStr != "-" {
		split := strings.Split(startStr, "-")

		if len(split) > 2 {
			return resptypes.SimpleError{Val: fmt.Errorf("ERR Start '%s' is not a valid stream entry ID! %s", startStr, c.getUsage())}
		}

		var err error
		start.Ms, err = strconv.ParseUint(split[0], 10, 64)
		if err != nil {
			return resptypes.SimpleError{Val: fmt.Errorf("ERR Could not parse ms part of start ID as an int64! %w", err)}
		}

		if len(split) == 2 {
			start.Seq, err = strconv.ParseUint(split[1], 10, 64)
			if err != nil {
				return resptypes.SimpleError{Val: fmt.Errorf("ERR Could not parse seq part of start ID as an int64! %w", err)}
			}
		}
	}

	end := redistypes.StreamEntryId{
		Ms:  redistypes.MaxSequenceNum,
		Seq: redistypes.MaxSequenceNum,
	}

	outer := make(resptypes.Array[resptypes.Array[resptypes.RespSerializable]], 1)
	inner := make(resptypes.Array[resptypes.RespSerializable], 2)
	inner[0] = streamKey
	inner[1] = dsVal.Stream.GetEntries(start, end)
	outer[0] = inner

	return outer
}
