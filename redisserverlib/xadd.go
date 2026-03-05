package redisserverlib

import (
	"context"
	"fmt"
	"strings"
)

type (
	xadd struct{}
)

func (c xadd) getUsage(ctx context.Context) string {
	return `
usage:
	XADD key field value [field value ...]
summary:
	Appends the specified stream entry to the stream at the specified key.
	If the key does not exist, XADD will create a new key with the given stream value as a side effect of running this command.
` + "\r\n"
}

func (c xadd) execute(ctx context.Context, r *redisCommandProcessor, params commandParams) commandResult {
	if len(params) < 4 {
		return fmt.Sprintf("-ERR XADD requires at least 4 arguments! %s", c.getUsage(ctx))
	}

	if len(params)%2 == 0 {
		return fmt.Sprintf("-ERR XADD requires an odd number of arguments! %s", c.getUsage(ctx))
	}

	streamKey := params[1]
	entryId := params[2]

	if strings.Compare(entryId, "0-0") <= 0 {
		return "-ERR The ID specified in XADD must be greater than 0-0\r\n"
	}

	if strings.Compare(entryId, r.LastEntryId) <= 0 {
		return "-ERR The ID specified in XADD is equal or smaller than the target stream top item\r\n"
	}

	streamAny := r.dataStore.GetOrCreate(streamKey, newRedisStreamAny, 0)
	if stream, ok := streamAny.(redisType_Stream); ok {

		if _, exists := stream.Get(entryId); exists {
			return fmt.Sprintf("-ERR Entry with ID %s already exists in stream! %s", entryId, c.getUsage(ctx))
		}

		fields := stream.GetOrCreate(entryId, newRedisStreamEntry, 0)
		for i := 3; i < len(params); i += 2 {
			field := params[i]
			value := params[i+1]
			fields.PushBack(redisType_FieldValuePair{Field: field, Value: value})
		}

		r.LastEntryId = entryId
		return fmt.Sprintf("$%d\r\n%s\r\n", len(entryId), entryId)
	}

	return fmt.Sprintf("-ERR XADD can only be called on streams! %s", c.getUsage(ctx))
}
