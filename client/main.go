package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/lib/logger"
	redisclientlib "github.com/codecrafters-io/redis-starter-go/lib/redis/client"
	redislib "github.com/codecrafters-io/redis-starter-go/lib/redis/common"
)

func main() {
	slog.SetDefault(slog.New(logger.NewHandler(slog.LevelDebug)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	network := "tcp4"
	host := "localhost"
	port := "6379"
	address := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout(network, address, 5*time.Second)
	if err != nil {
		slog.ErrorContext(ctx, "Dial failed!", "err", err, "conn", conn)
		return
	}

	defer conn.Close()
	redis := redisclientlib.NewRedisClient(conn)
	if len(os.Args) > 1 {
		request := strings.Join(os.Args[1:], " ")
		slog.DebugContext(ctx, "One shot mode", "request", request)
		redis.ExecuteCommand(ctx, request)
		return
	}

	slog.DebugContext(ctx, "Interactive mode")
	stdinChannel := redislib.ListenStdin(ctx, cancel)
	for {
		select {
		case <-ctx.Done():
			slog.DebugContext(ctx, "Main ctx cancelled")
			return
		case request := <-stdinChannel:
			result := redis.ExecuteCommand(ctx, request)
			if result.Err() != nil {
				slog.ErrorContext(ctx, "SendRequest returned an error!", "err", result.Err())
				continue
			}

			fmt.Println(result.Val())
		}
	}
}
