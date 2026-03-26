package redisserverlib

import (
	"context"
	"fmt"

	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	echo struct{}
)

func (c echo) moniker() string {
	return "ECHO"
}

func (c echo) getUsage() string {
	return `
usage:
	echo message
summary:
	Returns message.
`
}

func (c echo) execute(ctx context.Context, params commandParams) commandResult {
	if len(params) != 2 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR Unexpected number of params! %s", c.getUsage())}
	}

	return params[1]
}
