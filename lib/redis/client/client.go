package redisclientlib

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	redislib "github.com/codecrafters-io/redis-starter-go/lib/redis/common"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	redisClient struct {
		network string
		address string
		writer  io.Writer
		scanner *bufio.Scanner
		mu      sync.Mutex
	}

	RedisClient interface {
		ExecuteCommand(ctx context.Context, cmd string) CommandResult
	}
)

func NewRedisClient(readerWriter io.ReadWriter) RedisClient {
	redis := &redisClient{
		writer:  readerWriter,
		scanner: bufio.NewScanner(readerWriter),
	}
	redis.scanner.Split(redislib.ScanResp)

	if conn, ok := redis.writer.(net.Conn); ok {
		addr := conn.RemoteAddr()
		redis.network = addr.Network()
		redis.address = addr.String()
	}

	return redis
}

func (redis *redisClient) ExecuteCommand(ctx context.Context, command string) CommandResult {
	redis.mu.Lock()
	defer redis.mu.Unlock()

	if err := redis.ensureConnected(ctx); err != nil {
		return newCommandResult(nil, err)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	if dl, ok := ctx.Deadline(); ok {
		if conn, ok := redis.writer.(net.Conn); ok {
			conn.SetDeadline(dl)
			defer conn.SetDeadline(time.Time{})
		}
	}

	return redis.executeCommand(ctx, command)
}

func (redis *redisClient) executeCommand(ctx context.Context, command string) CommandResult {
	if command == "" {
		return newCommandResult(nil, errors.New("Cannot send empty command!"))
	}

	tokens := tokenizeCommandLine(command)
	if len(tokens) == 0 {
		return newCommandResult(nil, errors.New("Tokenization of command resulted in empty array!"))
	}

	bulkStringArr := resptypes.ToBulkStringArray(tokens)
	slog.DebugContext(ctx, "Converted tokens to bulk string array", "bulkStringArr", bulkStringArr)

	respStr := bulkStringArr.ToRespString()
	slog.DebugContext(ctx, "Serialized to resp string", "respStr", respStr)

	trySend := func() CommandResult {
		if redis.writer == nil {
			return newCommandResult(nil, errors.New("Writer is nil!"))
		}

		if redis.scanner == nil {
			return newCommandResult(nil, errors.New("Scanner is nil!"))
		}

		if _, err := redis.writer.Write([]byte(respStr)); err != nil {
			redis.close()
			return newCommandResult(nil, err)
		}

		slog.DebugContext(ctx, "Wrote resp string, scanning for text")
		if !redis.scanner.Scan() {
			redis.close()
			return newCommandResult(nil, redis.scanner.Err())
		}

		return newCommandResult(nil, nil)
	}

	result := trySend()
	if result.Err() != nil {
		slog.WarnContext(ctx, "Send/receive resulted in an error, attempting to reconnect.", "err", result.Err())
		if err := redis.ensureConnected(ctx); err != nil {
			return newCommandResult(nil, errors.Join(result.Err(), err))
		}

		retryResult := trySend()
		if retryResult.Err() != nil {
			return newCommandResult(nil, errors.Join(result.Err(), retryResult.Err()))
		}

		result = retryResult
	}

	text := redis.scanner.Text()
	slog.DebugContext(ctx, "Scanner read successful", "text", text)
	response, bytesCount := resptypes.ParseRespString(text)
	if bytesCount <= 0 {
		if err, ok := response.(resptypes.SimpleError); ok {
			return newCommandResult(err, err.Val)
		}

		return newCommandResult(response, errors.New("Parse error but received non-error type!"))
	}

	return newCommandResult(response, nil)
}

func (redis *redisClient) ensureConnected(ctx context.Context) error {
	if redis.writer == nil {
		if redis.network == "" {
			slog.WarnContext(ctx, "Network is empty, cannot reconnect!")
			return nil
		}

		if redis.address == "" {
			slog.WarnContext(ctx, "Address is empty, cannot reconnect!")
			return nil
		}

		conn, err := net.DialTimeout(redis.network, redis.address, 5*time.Second)
		if err != nil {
			return err
		}

		redis.writer = conn
		redis.scanner = bufio.NewScanner(conn)
		redis.scanner.Split(redislib.ScanResp)
	}

	return nil
}

func (redis *redisClient) close() error {
	if conn, ok := redis.writer.(net.Conn); ok {
		err := conn.Close()
		redis.writer = nil
		return err
	}
	return nil
}
