package commands

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/codecrafters-io/redis-starter-go/logger"
)

type (
	RESP2_Array          []string
	RESP2_CommandHandler func(RESP2_Array, chan string)
)

var (
	RESP2_Commands_Map = map[string]RESP2_CommandHandler{
		"PING":  PING,
		"ECHO":  ECHO,
		"SET":   SET,
		"GET":   GET,
		"RPUSH": RPUSH,
	}

	Cache       = map[string]string{}
	CacheMutex  sync.RWMutex
	Timers      = map[string]*time.Timer{}
	TimersMutex sync.Mutex
)

func RPUSH(tokens RESP2_Array, c chan string) {
	c <- "+PONG\r\n"
}

func PING(tokens RESP2_Array, c chan string) {
	c <- "+PONG\r\n"
}

func ECHO(tokens RESP2_Array, c chan string) {
	if len(tokens) < 2 {
		c <- "-ERR No message provided to ECHO!\r\n"
		return
	}
	if len(tokens) > 2 {
		c <- "-ERR ECHO accepts exactly 1 argument!\r\n"
		return
	}
	response := tokens[1]
	c <- fmt.Sprintf("$%d\r\n%s\r\n", len(response), response)
}

func SET(tokens RESP2_Array, c chan string) {
	arrSize := len(tokens)
	switch {
	case arrSize >= 3:
		key := tokens[1]
		value := tokens[2]

		// Validate key and value are not empty
		if key == "" {
			c <- "-ERR Key cannot be empty!\r\n"
			return
		}
		if value == "" {
			c <- "-ERR Value cannot be empty!\r\n"
			return
		}

		expiryDurationMs := 0
		err := error(nil)
		for i := 3; i < arrSize; i++ {
			if tokens[i] == "PX" {
				if i+1 >= arrSize {
					c <- "-ERR No expiration specified!\r\n"
					return
				} else {
					expiryDurationMs, err = strconv.Atoi(tokens[i+1])
					if err != nil {
						c <- fmt.Sprintf("-ERR Could not convert %s to an int for expiry! Err: %s\r\n", tokens[i+1], err)
						return
					}
				}
			}
		}

		// Stop any existing timer for this key
		TimersMutex.Lock()
		if timer, exists := Timers[key]; exists {
			logger.Debug("Cancelling existing timer", "key", key)
			timer.Stop()
			delete(Timers, key)
		}
		TimersMutex.Unlock()

		CacheMutex.Lock()
		Cache[key] = value
		CacheMutex.Unlock()
		logger.Debug("SET executed", "key", key, "value", value, "expiry_ms", expiryDurationMs)

		if expiryDurationMs > 0 {
			timer := time.NewTimer(time.Millisecond * time.Duration(expiryDurationMs))
			TimersMutex.Lock()
			Timers[key] = timer
			TimersMutex.Unlock()

			go func() {
				<-timer.C

				CacheMutex.Lock()
				delete(Cache, key)
				CacheMutex.Unlock()

				TimersMutex.Lock()
				delete(Timers, key)
				TimersMutex.Unlock()
				logger.Debug("Key expired", "key", key)
			}()
		}

		c <- "+OK\r\n"
	case arrSize == 2:
		c <- fmt.Sprintf("-ERR No value given for key %s!\r\n", tokens[1])
	case arrSize == 1:
		c <- "-ERR No key given!\r\n"
	}
}

func GET(tokens RESP2_Array, c chan string) {
	if len(tokens) < 2 {
		c <- "-ERR No key provided to GET!\r\n"
		return
	}
	key := tokens[1]
	CacheMutex.RLock()
	response, ok := Cache[key]
	CacheMutex.RUnlock()

	if ok {
		logger.Debug("GET cache hit", "key", key)
		c <- fmt.Sprintf("$%d\r\n%s\r\n", len(response), response)
	} else {
		logger.Debug("GET cache miss", "key", key)
		c <- "$-1\r\n"
	}
}
