package respcommands

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/codecrafters-io/redis-starter-go/resplib"
)

type (
	resp2_CommandInterface interface {
		getUsage(context.Context) string
		execute(context.Context, resplib.RESP2_CommandRequest) resplib.RESP2_CommandResponse
	}
)

var (
	resp2_Commands_Map = map[string]resp2_CommandInterface{
		// https://redis.io/docs/latest/commands/redis-8-4-commands/#quick-navigation

		"HELP": help{},

		// Connection commands: connectioncommands.go
		"PING": ping{},
		"ECHO": echo{},

		// String commands: stringcommands.go
		"SET": set{},
		"GET": get{},

		// List commands: listcommands.go
		"RPUSH":  rpush{},
		"LRANGE": lrange{},
		"LPUSH":  lpush{},
		"LLEN":   llen{},
		"LPOP":   lpop{},
		"BLPOP":  blpop{},
	}
)

func respifyArray(tokens []string) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "*%d\r\n", len(tokens))
	for _, token := range tokens {
		fmt.Fprintf(&buf, "$%d\r\n%s\r\n", len(token), token)
	}
	return buf.String()
}

func ExecuteCommand(ctx context.Context, request resplib.RESP2_CommandRequest) {
	respStr := respifyArray(request.Params)
	slog.DebugContext(ctx, "Command received", "request", respStr)

	entry, ok := resp2_Commands_Map[request.Params[0]]
	if !ok {
		request.ResponseChannel <- fmt.Sprintf("Unrecognized command '%s'!\r\n", request.Params[0])
		return
	}

	request.ResponseChannel <- entry.execute(ctx, request)
}
