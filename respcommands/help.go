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

func (c help) execute(ctx context.Context, request resplib.RESP2_CommandRequest) {
	if len(request.Params) != 2 {
		request.ResponseChannel <- c.getUsage(ctx)
		return
	}

	command, exists := resp2_Commands_Map[request.Params[1]]
	if !exists {
		request.ResponseChannel <- fmt.Sprintf("Command '%s' is not supported", request.Params[1])
		return
	}

	request.ResponseChannel <- command.getUsage(ctx)
}
