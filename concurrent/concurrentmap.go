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
	ConcurrentMap[Key comparable, Value any] struct {
		entries map[Key]mapEntry[Value]
		sync.RWMutex
	}
)

func NewConcurrentMap[K comparable, V any]() *ConcurrentMap[K, V] {
	return &ConcurrentMap[K, V]{
		entries: make(map[K]mapEntry[V]),
	}
}

func (m *ConcurrentMap[K, V]) Get(key K) (value V, exists bool) {
	m.RLock()
	defer m.RUnlock()
	entry, exists := m.entries[key]

	// Passive Expiration Check:
	// If an expiry is set (not Zero) and we are past that time,
	// pretend it's not there.
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return *new(V), false
	}

	return entry.data, exists
}

func (m *ConcurrentMap[K, V]) GetOrCreate(key K, newFunc func() V) V {
	// 1. FAST PATH: Uses m.Get's internal RLock
	if data, exists := m.Get(key); exists {
		return data
	}

	// 2. SLOW PATH: Full Write Lock
	m.Lock()
	defer m.Unlock()

	// 3. DOUBLE-CHECK: Re-evaluate state now that we hold the lock
	entry, exists := m.entries[key]
	isExpired := !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt)

	if exists && !isExpired {
		return entry.data
	}

	// 4. CLEANUP & CREATE:
	// If it existed but was expired, we overwrite it now.
	// If it didn't exist, we create it now.
	newValue := newFunc()
	m.entries[key] = mapEntry[V]{data: newValue}
	return newValue
}

func (m *ConcurrentMap[K, V]) Delete(key K) {
	m.Lock()
	defer m.Unlock()
	if entry, exists := m.entries[key]; exists {
		if entry.timer != nil {
			entry.timer.Stop()
		}
		delete(m.entries, key)
	}
}

func (m *ConcurrentMap[K, V]) Set(key K, value V, expiryDuration time.Duration) {
	m.Lock()
	defer m.Unlock()

	// 1. Clean up previous entry
	if previous, exists := m.entries[key]; exists && previous.timer != nil {
		previous.timer.Stop()
	}

	// 2. Prepare the new entry
	newEntry := mapEntry[V]{data: value}

	// 3. Handle expiration logic
	if expiryDuration > 0 {
		newEntry.expiresAt = time.Now().Add(expiryDuration)

		// Use AfterFunc to avoid manual goroutine management with a timer and a channel
		newEntry.timer = time.AfterFunc(expiryDuration, func() {
			// Double-check: only delete if this is still the same timer
			// (Prevents the "new value deleted by old timer" race)
			m.Lock()
			defer m.Unlock()
			if current, exists := m.entries[key]; exists && current.timer == newEntry.timer {
				delete(m.entries, key)
			}
		})
	}

	m.entries[key] = newEntry
}
