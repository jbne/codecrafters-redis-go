package respcommands

import (
	"context"
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/resplib"
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

func (c help) execute(ctx context.Context, request resplib.RESP2_CommandRequest) resplib.RESP2_CommandResponse {
	if len(request.Params) != 2 {
		return c.getUsage(ctx)
	}

	command, exists := resp2_Commands_Map[request.Params[1]]
	if !exists {
		return fmt.Sprintf("Command '%s' is not supported", request.Params[1])
	}

	return command.getUsage(ctx)
}
