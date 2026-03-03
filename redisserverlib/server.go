package redisserverlib

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/codecrafters-io/redis-starter-go/concurrent"
	"github.com/codecrafters-io/redis-starter-go/redislib"
)

type (
	commandParams    = redislib.RESP2_Array
	commandResult    = string
	commandInterface interface {
		getUsage(ctx context.Context) commandResult
		execute(ctx context.Context, r *redisCommandProcessor, params redislib.RESP2_Array) commandResult
	}

	redisDataStore struct {
		cache *concurrent.ConcurrentMap[string, string]
		lists *concurrent.ConcurrentMap[string, *concurrent.ConcurrentDeque[string]]
	}

	redisCommandProcessor struct {
		redisDataStore
		commands map[string]commandInterface
	}

	RedisCommandProcessor interface {
		ExecuteCommand(ctx context.Context, params redislib.RESP2_Array) commandResult
	}
)

func NewRedisCommandProcessor() RedisCommandProcessor {
	return &redisCommandProcessor{
		redisDataStore: redisDataStore{
			cache: concurrent.NewConcurrentMap[string, string](),
			lists: concurrent.NewConcurrentMap[string, *concurrent.ConcurrentDeque[string]](),
		},
		commands: map[string]commandInterface{
			// https://redis.io/docs/latest/commands/redis-8-6-commands/

			// Connection commands
			"PING": ping{},
			"ECHO": echo{},

			// String commands
			"SET": set{},
			"GET": get{},

			// List commands
			"RPUSH":  rpush{},
			"LRANGE": lrange{},
			"LPUSH":  lpush{},
			"LLEN":   llen{},
			"LPOP":   lpop{},
			"BLPOP":  blpop{},

			// Generic commands
			"HELP": help{},
			"TYPE": typeCmd{},
		},
	}
}

func respifyArray(tokens []string) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "*%d\r\n", len(tokens))
	for _, token := range tokens {
		fmt.Fprintf(&buf, "$%d\r\n%s\r\n", len(token), token)
	}
	return buf.String()
}

func (r *redisCommandProcessor) ExecuteCommand(ctx context.Context, params commandParams) commandResult {
	respStr := respifyArray(params)
	slog.DebugContext(ctx, "Command received", "request", respStr)

	entry, ok := r.commands[params[0]]
	if !ok {
		return fmt.Sprintf("Unrecognized command '%s'!\r\n", params[0])
	}

	return entry.execute(ctx, r, params)
}
