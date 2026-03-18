package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/lib/logger"
	redisclientlib "github.com/codecrafters-io/redis-starter-go/lib/redis/client"
	redislib "github.com/codecrafters-io/redis-starter-go/lib/redis/common"
)

func main() {
	slog.SetDefault(slog.New(logger.NewHandler(slog.LevelDebug)))
	ctx, cancel := context.WithCancel(context.Background())

	client, err := redisclientlib.NewRedisClient(ctx, "tcp4", "localhost:6379")
	if err != nil {
		slog.ErrorContext(ctx, "Error encountered during client creation!", "err", err)
		return
	}

	if len(os.Args) > 1 {
		request := strings.Join(os.Args[1:], " ")
		slog.DebugContext(ctx, "One shot mode", "request", request)
		client.SendRequest(ctx, request)
		return
	}

	slog.DebugContext(ctx, "Interactive mode")

	stdinChannel := make(chan string)
	var wg sync.WaitGroup
	wg.Go(func() {
		redislib.ListenStdin(ctx, stdinChannel)
		slog.DebugContext(ctx, "StdinWorker done")
		cancel()
	})
	defer wg.Wait()

	for {
		select {
		case <-ctx.Done():
		case <-client.Done():
			cancel()
			return
		case stdin := <-stdinChannel:
			response := client.SendRequest(ctx, stdin)
			fmt.Println(response)
		}
	}
}
