package concurrent

import (
	"context"
	"fmt"
	"sync"
)

type (
	ConcurrentDeque[T any] struct {
		mu     sync.Mutex
		buf    []T
		head   int
		tail   int
		count  int
		waiter chan struct{}
	}
)

func NewConcurrentDeque[T any]() *ConcurrentDeque[T] {
	q := &ConcurrentDeque[T]{
		buf: make([]T, 16), // Start with a small power-of-two capacity
	}
	return q
}

// PushBack: O(1) amortized
func (q *ConcurrentDeque[T]) PushBack(values ...T) int {
	q.mu.Lock()
	defer q.mu.Unlock()

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

	if q.waiter != nil {
		close(q.waiter)
	}

	return q.count
}

// PushFront: O(1) amortized
func (q *ConcurrentDeque[T]) PushFront(values ...T) int {
	q.mu.Lock()
	defer q.mu.Unlock()

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

	if q.waiter != nil {
		close(q.waiter)
	}

	return q.count
}

// PopFront: O(1)
func (q *ConcurrentDeque[T]) PopFront(n int) []T {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.popFrontNoLock(n)
}

func (q *ConcurrentDeque[T]) PopFrontAsync(ctx context.Context) (error, []T) {
	q.mu.Lock()

	if q.count > 0 {
		res := q.popFrontNoLock(1)
		q.mu.Unlock()
		return nil, res
	}

	if q.waiter != nil {
		q.mu.Unlock()
		return fmt.Errorf("already waiting"), nil
	}

	q.waiter = make(chan struct{})
	q.mu.Unlock()

	select {
	case <-ctx.Done():
		q.mu.Lock()
		if q.waiter != nil {
			close(q.waiter)
			q.waiter = nil
		}
		q.mu.Unlock()
		return ctx.Err(), nil
	case <-q.waiter:
		q.mu.Lock()
		res := q.popFrontNoLock(1)
		q.waiter = nil
		q.mu.Unlock()
		return nil, res
	}
}

func (q *ConcurrentDeque[T]) GetRange(startIndex int, stopIndex int) []T {
	q.mu.Lock()
	defer q.mu.Unlock()

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
	q.mu.Lock()
	defer q.mu.Unlock()
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
