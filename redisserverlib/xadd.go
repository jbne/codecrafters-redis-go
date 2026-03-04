package redisserverlib

import (
	"context"
	"fmt"
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

	streamAny := r.dataStore.GetOrCreate(streamKey, newRedisStreamAny, 0)
	if _, ok := streamAny.(redisType_Stream); ok {
		return fmt.Sprintf("$%d\r\n%s\r\n", len(entryId), entryId)
	}

	return fmt.Sprintf("-ERR XADD can only be called on streams! %s", c.getUsage(ctx))
}
