package redisclientlib

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	redislib "github.com/codecrafters-io/redis-starter-go/lib/redis/common"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	redisClient struct {
		done            chan struct{}
		conn            net.Conn
		network         string
		address         string
		requestChannel  chan string
		responseChannel chan resptypes.BaseInterface
	}

	RedisClient interface {
		Done() <-chan struct{}
		SendRequest(ctx context.Context, request string) (response resptypes.BaseInterface)
		connect(ctx context.Context) (err error)
		disconnect(ctx context.Context) (err error)
	}
)

func NewRedisClient(ctx context.Context, network string, address string) (RedisClient, error) {
	client := &redisClient{
		done:    make(chan struct{}),
		network: network,
		address: address,
	}
	if err := client.connect(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *redisClient) Done() <-chan struct{} {
	return c.done
}

func (c *redisClient) SendRequest(ctx context.Context, request string) resptypes.BaseInterface {
	c.requestChannel <- request
	return <-c.responseChannel
}

func (c *redisClient) connect(ctx context.Context) error {
	slog.With("address", c.address).InfoContext(ctx, "Connecting to server")
	conn, err := net.Dial(c.network, c.address)

	if err != nil {
		slog.ErrorContext(ctx, "Failed to connect", "err", err)
		return err
	}

	c.conn = conn
	c.requestChannel = make(chan string)
	c.responseChannel = make(chan resptypes.BaseInterface)
	go func() {
		ctx, cancel := context.WithCancel(ctx)
		slog.InfoContext(ctx, "Connected to server")
		defer c.disconnect(ctx)
		defer close(c.done)

		var wg sync.WaitGroup
		wg.Go(func() {
			writeWorker(ctx, conn, c.requestChannel)
			slog.DebugContext(ctx, "WriteWorker done")
			cancel()
		})
		wg.Go(func() {
			in, ctx := redislib.CreateScannerChannel(ctx, conn, redislib.ScanResp)
			readWorker(ctx, in, c.responseChannel)
			slog.DebugContext(ctx, "ReadWorker done")
			cancel()
		})
		wg.Wait()
	}()

	return err
}

func (c *redisClient) disconnect(ctx context.Context) (err error) {
	if c.conn != nil {
		err = c.conn.Close()
		if err != nil {
			slog.ErrorContext(ctx, "Error during disconnect!", "err", err)
		}
	}

	return err
}

func writeWorker(ctx context.Context, conn net.Conn, requestChannel <-chan string) {
	var buf bytes.Buffer
	for {
		select {
		case <-ctx.Done():
			slog.DebugContext(ctx, "writeWorker context cancelled")
			return
		case input := <-requestChannel:
			tokens := tokenizeCommandLine(input)
			if len(tokens) == 0 {
				slog.WarnContext(ctx, "No tokens parsed from input", "input", input)
				continue
			}

			buf.Reset()
			fmt.Fprintf(&buf, "*%d\r\n", len(tokens))
			for _, token := range tokens {
				fmt.Fprintf(&buf, "$%d\r\n%s\r\n", len(token), token)
			}

			_, err := conn.Write(buf.Bytes())
			if err != nil {
				slog.ErrorContext(ctx, "Failed to write command", "error", err)
				return
			}

			slog.DebugContext(ctx, "Command sent", "request", buf.String())
		}
	}
}

func readWorker(ctx context.Context, in <-chan string, out chan<- resptypes.BaseInterface) {
	for {
		select {
		case <-ctx.Done():
			slog.DebugContext(ctx, "readWorker context cancelled")
			return
		case respStr := <-in:
			respType, bytesCount := resptypes.ParseRespString(respStr)
			if bytesCount == 0 {
				slog.ErrorContext(ctx, "Received invalid RESP response!", "respStr", respStr, "respType", respType)
				return
			}
			out <- respType
		}
	}
}
