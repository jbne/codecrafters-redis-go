package redisserverlib

import (
	"context"
	"fmt"
)

type (
	help struct {
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
` + "\r\n"
}

func (c help) execute(ctx context.Context, r *redisCommandProcessor, params commandParams) commandResult {
	if len(params) != 2 {
		return c.getUsage(ctx)
	}

	command, exists := r.commands[params[1]]
	if !exists {
		return fmt.Sprintf("Command '%s' is not supported", params[1])
	}

	return command.getUsage(ctx)
}
