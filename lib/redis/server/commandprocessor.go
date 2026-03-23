package redisserverlib

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/codecrafters-io/redis-starter-go/lib/concurrent"
	redistypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/redis"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	redisDataStore struct {
		dataStore         concurrent.ConcurrentMap[string, any]
		lastStreamEntryId redistypes.StreamEntryId
	}

	redisCommandProcessor struct {
		redisDataStore
		commands map[string]commandDefinition
	}
)

func NewRedisCommandProcessor() CommandProcessor {
	redisDataStore := redisDataStore{
		dataStore: concurrent.NewConcurrentMap[string, any](),
		lastStreamEntryId: redistypes.StreamEntryId{
			Id:  "0-0",
			Ms:  0,
			Seq: 0,
		},
	}

	// https://redis.io/docs/latest/commands/redis-8-6-commands/
	commands := map[string]commandDefinition{}

	// Connection commands
	commands["PING"] = ping{}
	commands["ECHO"] = echo{}

	// // String commands
	commands["SET"] = set{&redisDataStore}
	commands["GET"] = get{&redisDataStore}

	// // List commands
	commands["RPUSH"] = rpush{&redisDataStore}
	commands["LRANGE"] = lrange{&redisDataStore}
	commands["LPUSH"] = lpush{&redisDataStore}
	commands["LLEN"] = llen{&redisDataStore}
	commands["LPOP"] = lpop{&redisDataStore}
	commands["BLPOP"] = blpop{&redisDataStore}

	// // Stream commands
	commands["XADD"] = xadd{&redisDataStore}
	commands["XRANGE"] = xrange{&redisDataStore}

	// // Generic commands
	commands["TYPE"] = typeCmd{&redisDataStore}
	commands["HELP"] = help{commands}

	return &redisCommandProcessor{
		redisDataStore: redisDataStore,
		commands:       commands,
	}
}

func (r *redisCommandProcessor) ExecuteCommand(ctx context.Context, respStr string) resptypes.BaseInterface {
	slog.DebugContext(ctx, "Command received", "respStr", respStr)

	parsed, byteCount := resptypes.ParseRespString(respStr)
	if byteCount == 0 {
		return parsed
	}

	respArr, ok := parsed.(resptypes.Array[resptypes.BaseInterface])
	if !ok {
		return resptypes.SimpleError{Val: fmt.Errorf("NOTEXPECTED Parsed request was not a RESP array! Got: %v", respStr)}
	}

	bulkStrings := make(resptypes.Array[resptypes.BulkString], 0)
	for _, e := range respArr {
		switch e := e.(type) {
		case resptypes.BulkString:
			bulkStrings = append(bulkStrings, e)
			continue
		default:
			return resptypes.SimpleError{Val: fmt.Errorf("NOTEXPECTED Parsed request was not an array of bulk strings! Got: %v", respStr)}
		}
	}

	commandName := bulkStrings[0].Val
	entry, ok := r.commands[commandName]
	if !ok {
		return resptypes.SimpleError{Val: fmt.Errorf("NOTSUPPORTED Command '%s' is not supported!", commandName)}
	}

	result := entry.execute(ctx, bulkStrings)
	return result
}
