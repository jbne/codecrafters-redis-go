package concurrent

import (
	"container/list"
	"sync"
	"time"
)

type (
	ConcurrentDeque[T any] struct {
		mu      sync.RWMutex
		buf     []T
		head    int
		tail    int
		count   int
		waiters list.List
	}
)

func NewConcurrentDeque[T any]() *ConcurrentDeque[T] {
	return &ConcurrentDeque[T]{
		buf: make([]T, 16), // Start with a small power-of-two capacity
	}
}

// PushBack: O(1) amortized
func (q *ConcurrentDeque[T]) PushBack(values ...T) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	defer q.notifyAwaiters(values...)

	// Ensure we have enough space for any items that might end up in the buffer
	targetCount := q.count + len(values)
	q.resizeIfNecessaryNoLock(targetCount)

	mask := len(q.buf) - 1

	// Process each value one by one
	for _, val := range values {
		q.buf[q.tail] = val
		q.tail = (q.tail + 1) & mask
		q.count++
	}

	return q.count
}

func (q *ConcurrentDeque[T]) notifyAwaiters(values ...T) {
	for q.waiters.Len() > 0 && len(values) > 0 {
		q.waiters.Front().Value.(chan []T) <- values[0:1]
		q.waiters.Remove(q.waiters.Front())
		values = values[1:]
	}
}

// PushFront: O(1) amortized
func (q *ConcurrentDeque[T]) PushFront(values ...T) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	defer q.notifyAwaiters(values...)

	// Ensure we have enough space for any items that might end up in the buffer
	targetCount := q.count + len(values)
	q.resizeIfNecessaryNoLock(targetCount)

	mask := len(q.buf) - 1

	// Process each value one by one
	for _, val := range values {
		q.head = (q.head - 1) & mask
		q.buf[q.head] = val
		q.count++
	}

	return q.count
}

// PopFront: O(1)
func (q *ConcurrentDeque[T]) PopFront(n int) []T {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.popFrontNoLock(n)
}

func (q *ConcurrentDeque[T]) PopFrontAsync(timeout time.Duration) <-chan []T {
	waiter := make(chan []T, 1)

	q.mu.Lock()
	if q.count > 0 {
		val := q.popFrontNoLock(1)
		q.mu.Unlock()
		waiter <- val
		return waiter
	}

	q.waiters.PushBack(waiter)
	q.mu.Unlock()

	return waiter
}

func (q *ConcurrentDeque[T]) GetRange(startIndex int, stopIndex int) []T {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.count == 0 {
		return nil
	}

	// Convert Redis negative indices
	if startIndex < 0 {
		startIndex = q.count + startIndex
	}
	if stopIndex < 0 {
		stopIndex = q.count + stopIndex
	}

	// Clamp to valid range
	startIndex = max(0, startIndex)
	stopIndex = min(q.count-1, stopIndex)

	// Return nil if the range is inverted
	if startIndex > stopIndex {
		return []T{}
	}

	// Create the result slice
	n := stopIndex - startIndex + 1
	result := make([]T, n)
	mask := len(q.buf) - 1

	// UNROLL the ring buffer
	for i := range n {
		// Logical Index (startIndex + i) -> Physical Index
		physicalIdx := (q.head + startIndex + i) & mask
		result[i] = q.buf[physicalIdx]
	}

	return result
}

func (q *ConcurrentDeque[T]) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.count
}

// All functions below this line are intended to be used when the mutex is acquired by the caller

// Internal Resize: O(N) but only happens when doubling capacity
func (q *ConcurrentDeque[T]) resizeIfNecessaryNoLock(targetCount int) {
	if targetCount <= len(q.buf) {
		return
	}

	newSize := max(len(q.buf), 1)
	for newSize < targetCount {
		newSize <<= 1
	}

	newBuf := make([]T, newSize)
	if q.count > 0 { // Only copy if there's data!
		oldMask := len(q.buf) - 1
		for i := 0; i < q.count; i++ {
			newBuf[i] = q.buf[(q.head+i)&oldMask]
		}
	}

	q.buf = newBuf
	q.head = 0
	q.tail = q.count
}

func (q *ConcurrentDeque[T]) popFrontNoLock(n int) []T {
	n = max(0, min(n, q.count))
	res := make([]T, n)
	for i := 0; i < n; i++ {
		res[i] = q.buf[q.head]
		q.buf[q.head] = *new(T) // Zero out for GC
		mask := len(q.buf) - 1
		q.head = (q.head + 1) & mask
		q.count--
	}
	return res
}
