package respcommands

import (
	"context"
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/resplib"
)

type (
	ping struct{}
	echo struct{}
)

func (c ping) getUsage(ctx context.Context) string {
	return `
usage:
	PING [message]
summary:
	Returns PONG if no argument is provided, otherwise return a copy of the argument as a bulk. This command is useful for:
	1. Testing whether a connection is still alive.
	2. Verifying the server's ability to serve data - an error is returned when this isn't the case (e.g., during load from persistence or accessing a stale replica).
	3. Measuring latency.

	If the client is subscribed to a channel or a pattern, it will instead return a multi-bulk with a "pong" in the first position and an empty bulk in the second position, unless an argument is provided in which case it returns a copy of the argument.
` + "\r\n"
}

func (c ping) execute(ctx context.Context, request resplib.RESP2_CommandRequest) resplib.RESP2_CommandResponse {
	return "+PONG\r\n"
}

func (c echo) getUsage(ctx context.Context) string {
	return `
usage:
	echo message
summary:
	Returns message.
` + "\r\n"
}

func (c echo) execute(ctx context.Context, request resplib.RESP2_CommandRequest) resplib.RESP2_CommandResponse {
	if len(request.Params) != 2 {
		return fmt.Sprintf("-ERR Unexpected number of params! %s", c.getUsage(ctx))
	}

	response := request.Params[1]
	return fmt.Sprintf("$%d\r\n%s\r\n", len(response), response)
}
