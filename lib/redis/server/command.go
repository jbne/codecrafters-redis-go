package redisserverlib

import (
	"context"

	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	commandParams = resptypes.Array[resptypes.BulkString]
	commandResult = resptypes.BaseInterface

	commandUsage interface {
		getUsage(ctx context.Context) string
	}

	commandDefinition interface {
		commandUsage
		execute(ctx context.Context, params commandParams) commandResult
	}

	CommandProcessor interface {
		ExecuteCommand(ctx context.Context, respStr string) resptypes.BaseInterface
	}
)
