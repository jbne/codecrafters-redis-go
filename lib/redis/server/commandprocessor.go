package redisserverlib

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	redistypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/redis"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	commandParams = resptypes.Array[resptypes.BulkString]
	commandResult = resptypes.RespSerializable

	commandUsage interface {
		getUsage() string
	}

	commandDefinition interface {
		commandUsage
		execute(ctx context.Context, params commandParams) commandResult
		moniker() string
	}

	commandMap map[string]commandDefinition

	redisCommandProcessor struct {
		ds       redistypes.DataStore
		commands commandMap
	}

	CommandProcessor interface {
		ExecuteCommand(ctx context.Context, respStr string) resptypes.RespSerializable
	}
)

func (m *commandMap) registerCommand(cd commandDefinition) {
	(*m)[cd.moniker()] = cd
}

func NewRedisCommandProcessor() CommandProcessor {
	redisDataStore := redistypes.NewRedisDataStore()

	// https://redis.io/docs/latest/commands/redis-8-6-commands/
	commands := make(commandMap)

	// Connection commands
	commands.registerCommand(ping{})
	commands.registerCommand(echo{})

	// String commands
	commands.registerCommand(set{redisDataStore})
	commands.registerCommand(get{redisDataStore})

	// List commands
	commands.registerCommand(rpush{redisDataStore})
	commands.registerCommand(lrange{redisDataStore})
	commands.registerCommand(lpush{redisDataStore})
	commands.registerCommand(llen{redisDataStore})
	commands.registerCommand(lpop{redisDataStore})
	commands.registerCommand(blpop{redisDataStore})

	// Stream commands
	commands.registerCommand(xadd{redisDataStore})
	commands.registerCommand(xrange{redisDataStore})
	commands.registerCommand(xread{redisDataStore})

	// Generic commands
	commands.registerCommand(typeCmd{redisDataStore})
	commands.registerCommand(help{commands})

	return &redisCommandProcessor{
		ds:       redisDataStore,
		commands: commands,
	}
}

func (r *redisCommandProcessor) ExecuteCommand(ctx context.Context, respStr string) resptypes.RespSerializable {
	slog.DebugContext(ctx, "Command received", "respStr", respStr)

	parsed, byteCount := resptypes.ParseRespString(respStr)
	if byteCount == 0 {
		return parsed
	}

	respArr, ok := parsed.(resptypes.Array[resptypes.RespSerializable])
	if !ok {
		return resptypes.SimpleError{Val: fmt.Errorf("NOTEXPECTED Parsed request was not a RESP array! Got: %v", respStr)}
	}

	if len(respArr) <= 0 {
		return resptypes.SimpleError{Val: fmt.Errorf("NOTEXPECTED Array is empty! Got: %d", len(respArr))}
	}

	bulkStrings := make(resptypes.Array[resptypes.BulkString], len(respArr))
	for i, e := range respArr {
		switch e := e.(type) {
		case resptypes.BulkString:
			bulkStrings[i] = e
			continue
		default:
			return resptypes.SimpleError{Val: fmt.Errorf("NOTEXPECTED Parsed request was not an array of bulk strings! Got: %v", respStr)}
		}
	}

	commandName := bulkStrings[0].Val
	commandName = strings.ToUpper(commandName)
	entry, ok := r.commands[commandName]
	if !ok {
		return resptypes.SimpleError{Val: fmt.Errorf("NOTSUPPORTED Command '%s' is not supported!", commandName)}
	}

	result := entry.execute(ctx, bulkStrings)
	return result
}
