package redisserverlib

import (
	"context"
	"fmt"

	redistypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/redis"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	get struct {
		redistypes.DataStore
	}
)

func (c get) moniker() string {
	return "GET"
}

func (c get) getUsage() string {
	return `
usage:
	get key
summary:
	Get the value of key. If the key does not exist the special value nil is returned.
	An error is returned if the value stored at key is not a string, because GET only handles string values.
`
}

func (c get) execute(ctx context.Context, params commandParams) commandResult {
	if len(params) < 2 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR No key provided to GET!")}
	}

	key := params[1].Val
	if dsVal, exists := c.Get(key); exists {
		if dsVal.Type != redistypes.TypeString {
			return resptypes.SimpleError{Val: fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")}
		}

		return dsVal.String
	}

	return resptypes.BulkString{Length: -1}
}
