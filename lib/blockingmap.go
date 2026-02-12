package lib

import (
	"sync"
	"time"

	"github.com/codecrafters-io/redis-starter-go/logger"
)

type (
	BlockingMap[Key comparable, Value any] struct {
		values map[Key]Value
		mapMutex sync.RWMutex

		timers map[Key]time.Timer
		timersMutex sync.Mutex
	}
)

func (m *BlockingMap[K, V]) Get(key K) (V, bool) {
	m.mapMutex.RLock()
	value, ok := m.values[key]
	m.mapMutex.RUnlock()
	return value, ok
}

func (m *BlockingMap[K, V]) Delete(key K) {
	m.mapMutex.Lock()
	delete(m.values, key)
	m.mapMutex.Unlock()
}

func (m *BlockingMap[K, V]) Set(key K, value V, expiryDuration time.Duration) {
	// Stop any existing timer for this key
	m.timersMutex.Lock()
	if timer, exists := m.timers[key]; exists {
		logger.Debug("Cancelling existing timer", "key", key)
		timer.Stop()
		delete(m.timers, key)
	}
	m.timersMutex.Unlock()
	m.mapMutex.Lock()
	m.values[key] = value
	m.mapMutex.Unlock()
	logger.Debug("SET executed", "key", key, "value", value, "expiry_ms", expiryDuration.Milliseconds())

	if expiryDuration.Milliseconds() > 0 {
		logger.Debug("Setting expiry timer", "key", key, "duration_ms", expiryDuration.Milliseconds())
		timer := time.NewTimer(expiryDuration)
		m.timersMutex.Lock()
		m.timers[key] = *timer
		m.timersMutex.Unlock()

		go func() {
			<-timer.C

			m.mapMutex.Lock()
			delete(m.values, key)
			m.mapMutex.Unlock()

			m.timersMutex.Lock()
			delete(m.timers, key)
			m.timersMutex.Unlock()
			logger.Debug("Key expired", "key", key)
		}()
	}
}

func NewBlockingMap[K comparable, V any]() *BlockingMap[K, V] {
	return &BlockingMap[K, V]{
		values: make(map[K]V),
		timers: make(map[K]time.Timer),
	}
}