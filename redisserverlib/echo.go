package redisserverlib

import (
	"context"
	"fmt"
)

type (
	echo struct{}
)

func (c echo) getUsage(ctx context.Context) string {
	return `
usage:
	echo message
summary:
	Returns message.
` + "\r\n"
}

func (c echo) execute(ctx context.Context, r *redisCommandProcessor, params commandParams) commandResult {
	if len(params) != 2 {
		return fmt.Sprintf("-ERR Unexpected number of params! %s", c.getUsage(ctx))
	}

	response := params[1]
	return fmt.Sprintf("$%d\r\n%s\r\n", len(response), response)
}
