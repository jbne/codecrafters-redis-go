package redisserverlib

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	redistypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/redis"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

var (
	xrangeEntryIdRegexp = regexp.MustCompile(`^\d+(-\d+)?$`)
)

type (
	xrange struct {
		*redisDataStore
	}
)

func (c xrange) getUsage(ctx context.Context) string {
	return `
usage:
	XRANGE key start end [COUNT count]
summary:
	The command returns the stream entries matching a given range of IDs.
	The range is specified by a minimum and maximum ID.
	All the entries having an ID between the two specified or exactly one of the two IDs specified (closed interval) are returned.
` + "\r\n"
}

func (c xrange) execute(ctx context.Context, params commandParams) commandResult {
	if len(params) != 4 {
		return resptypes.Error{Val: fmt.Errorf("ERR XRANGE requires exactly 4 arguments! %s", c.getUsage(ctx))}
	}

	streamKey := params[1].Val
	start := params[2].Val
	end := params[3].Val

	if !xrangeEntryIdRegexp.MatchString(start) {
		return resptypes.Error{Val: fmt.Errorf("ERR Start '%s' is not a valid stream entry ID! %s", start, c.getUsage(ctx))}
	}

	if !xrangeEntryIdRegexp.MatchString(end) {
		return resptypes.Error{Val: fmt.Errorf("ERR End '%s' is not a valid stream entry ID! %s", end, c.getUsage(ctx))}
	}

	if !strings.Contains(start, "-") {
		start = start + "-0"
	}

	if !strings.Contains(end, "-") {
		// This is the max possible sequence number that Redis supports.
		end = end + "-18446744073709551615"
	}

	streamAny := c.dataStore.GetOrCreate(streamKey, redistypes.NewStream, 0)
	stream, ok := streamAny.(redistypes.Stream)
	if !ok {
		return resptypes.Error{Val: fmt.Errorf("ERR XRANGE can only be called on streams! %s", c.getUsage(ctx))}
	}

	arr := make(resptypes.Array[resptypes.BaseType], 0)
	stream.ForEach(func(entryId string, entries redistypes.StreamEntries) {
		if start <= entryId && entryId <= end {
			inner := make(resptypes.Array[resptypes.BaseType], 0)
			inner = append(inner, resptypes.NewBulkString(entryId))
			entries.ForEach(func(_ int, item redistypes.StreamEntry) {
				inner = append(inner, item)
			})

			arr = append(arr, inner)
		}
	})

	slog.DebugContext(ctx, "", "arr", arr.ToRespString())
	return arr
}
