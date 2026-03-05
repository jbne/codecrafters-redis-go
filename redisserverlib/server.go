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

	redisType_List   = *concurrent.ConcurrentDeque[string]
	redisType_String = string

	redisType_FieldValuePair struct {
		Field string
		Value string
	}

	redisType_StreamEntry = *concurrent.ConcurrentDeque[redisType_FieldValuePair]

	// Entry ID => [field => value, ...]
	redisType_Stream = *concurrent.ConcurrentMap[string, redisType_StreamEntry]

	redisDataStore struct {
		dataStore   *concurrent.ConcurrentMap[string, any]
		LastEntryId string
	}

	redisCommandProcessor struct {
		redisDataStore
		commands map[string]commandInterface
	}

	RedisCommandProcessor interface {
		ExecuteCommand(ctx context.Context, params redislib.RESP2_Array) commandResult
	}
)

func newRedisStreamEntry() redisType_StreamEntry {
	return concurrent.NewConcurrentDeque[redisType_FieldValuePair]()
}

func newRedisStreamAny() any {
	return concurrent.NewConcurrentMap[string, redisType_StreamEntry]()
}

func newRedisListAny() any {
	return concurrent.NewConcurrentDeque[string]()
}

func respifyArray(tokens []string) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "*%d\r\n", len(tokens))
	for _, token := range tokens {
		fmt.Fprintf(&buf, "$%d\r\n%s\r\n", len(token), token)
	}
	return buf.String()
}

func NewRedisCommandProcessor() RedisCommandProcessor {
	return &redisCommandProcessor{
		redisDataStore: redisDataStore{
			dataStore:   concurrent.NewConcurrentMap[string, any](),
			LastEntryId: "0-0",
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

			// Stream commands
			"XADD": xadd{},

			// Generic commands
			"HELP": help{},
			"TYPE": typeCmd{},
		},
	}
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
