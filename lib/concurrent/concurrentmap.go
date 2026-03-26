package concurrent

import (
	"sync"
	"time"
)

type (
	mapEntry[Value any] struct {
		data      Value
		timer     *time.Timer
		expiresAt time.Time
	}

	ConcurrentMap[K any, V any] interface {
		Get(key K) (value V, exists bool)
		Delete(key K)
		GetOrCreate(key K, newFunc func() V, expiryDuration time.Duration) V
	}

	concurrentMap[Key comparable, Value any] struct {
		entries map[Key]mapEntry[Value]
		mu      sync.RWMutex
	}
)

func NewConcurrentMap[K comparable, V any]() ConcurrentMap[K, V] {
	return &concurrentMap[K, V]{
		entries: make(map[K]mapEntry[V]),
	}
}

func (m *concurrentMap[K, V]) Get(key K) (value V, exists bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, exists := m.entries[key]

	// Passive Expiration Check:
	// If an expiry is set (not Zero) and we are past that time,
	// pretend it's not there.
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		var zero V
		return zero, false
	}

	return entry.data, exists
}

func (m *concurrentMap[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry, exists := m.entries[key]; exists {
		if entry.timer != nil {
			entry.timer.Stop()
		}
		delete(m.entries, key)
	}
}

func (m *concurrentMap[K, V]) GetOrCreate(key K, newFunc func() V, expiryDuration time.Duration) V {
	// 1. FAST PATH: Uses m.Get's internal RLock
	if data, exists := m.Get(key); exists {
		return data
	}

	// 2. SLOW PATH: Full Write Lock
	m.mu.Lock()
	defer m.mu.Unlock()

	// 3. DOUBLE-CHECK: Re-evaluate state now that we hold the lock
	entry, exists := m.entries[key]
	isExpired := !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt)

	if exists {
		if !isExpired {
			return entry.data
		}

		// If it exists but is expired, we should clean it up before creating a new one.
		if entry.timer != nil {
			entry.timer.Stop()
		}
	}

	// 4. CLEANUP & CREATE:
	// If it existed but was expired, we overwrite it now.
	// If it didn't exist, we create it now.
	newValue := newFunc()
	newEntry := mapEntry[V]{data: newValue}

	// 5. Handle expiration logic
	if expiryDuration > 0 {
		newEntry.expiresAt = time.Now().Add(expiryDuration)

		// Use AfterFunc to avoid manual goroutine management with a timer and a channel
		newEntry.timer = time.AfterFunc(expiryDuration, func() {
			// Double-check: only delete if this is still the same timer
			// (Prevents the "new value deleted by old timer" race)
			m.mu.Lock()
			defer m.mu.Unlock()
			if current, exists := m.entries[key]; exists && current.timer == newEntry.timer {
				delete(m.entries, key)
			}
		})
	}

	m.entries[key] = newEntry
	return newValue
}
