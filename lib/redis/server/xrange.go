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
	xrange struct {
		redistypes.DataStore
	}
)

func (c xrange) moniker() string {
	return "XRANGE"
}

func (c xrange) getUsage() string {
	return `
usage:
	XRANGE key start end [COUNT count]
summary:
	The command returns the stream entries matching a given range of IDs.
	The range is specified by a minimum and maximum ID.
	All the entries having an ID between the two specified or exactly one of the two IDs specified (closed interval) are returned.
`
}

func (c xrange) execute(ctx context.Context, params commandParams) commandResult {
	if len(params) != 4 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR XRANGE requires exactly 4 arguments! %s", c.getUsage())}
	}

	streamKey := params[1].Val
	dsVal, exists := c.Get(streamKey)
	if !exists {
		return resptypes.Array[resptypes.RespSerializable]{}
	}

	if dsVal.Type != redistypes.TypeStream {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR XRANGE can only be called on streams! %s", c.getUsage())}
	}

	startStr := params[2].Val
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

	endStr := params[3].Val
	end := redistypes.StreamEntryId{}
	if endStr == "+" {
		end.Ms = redistypes.MaxSequenceNum
		end.Seq = redistypes.MaxSequenceNum
	} else {
		split := strings.Split(endStr, "-")

		if len(split) > 2 {
			return resptypes.SimpleError{Val: fmt.Errorf("ERR End '%s' is not a valid stream entry ID! %s", endStr, c.getUsage())}
		}

		var err error
		end.Ms, err = strconv.ParseUint(split[0], 10, 64)
		if err != nil {
			return resptypes.SimpleError{Val: fmt.Errorf("ERR Could not parse ms part of end ID as an int64! %w", err)}
		}

		if len(split) == 2 {
			end.Seq, err = strconv.ParseUint(split[1], 10, 64)
			if err != nil {
				return resptypes.SimpleError{Val: fmt.Errorf("ERR Could not parse seq part of end ID as an int64! %w", err)}
			}
		}
	}

	return dsVal.Stream.GetEntries(start, end)
}
