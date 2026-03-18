package redisserverlib

import (
	"context"
	"fmt"

	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	help struct {
		usages map[string]commandDefinition
	}
)

func (c help) getUsage(ctx context.Context) string {
	return `
usage:
	HELP (<commandname> | @<category>)

summary:
	HELP <commandname> shows specific help for the command given as argument.
	HELP @<category> shows all the commands about a given category.

	Only HELP <commandname> is currently implemented.
`
}

func (c help) execute(ctx context.Context, params commandParams) commandResult {
	if len(params) != 2 {
		return resptypes.SimpleString{Val: c.getUsage(ctx)}
	}

	commandName := params[1].Val
	command, exists := c.usages[commandName]
	if !exists {
		return resptypes.SimpleError{Val: fmt.Errorf("NOTSUPPORTED Command '%s' is not supported", commandName)}
	}

	return resptypes.SimpleString{Val: command.getUsage(ctx)}
}
