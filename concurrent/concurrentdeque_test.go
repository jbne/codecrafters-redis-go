package concurrent_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/concurrent"
)

type (
	Assertion func(i int, t *testing.T, q *concurrent.ConcurrentDeque[any])

	TestCase struct {
		name  string
		steps []Assertion
	}
)

func assertLen(expected int) Assertion {
	return func(i int, t *testing.T, q *concurrent.ConcurrentDeque[any]) {
		if actual := q.Len(); actual != expected {
			t.Errorf("i: %d, Len() = %v; Expected: %v", i, actual, expected)
		}
	}
}

func assertPushBack(expected int, values ...any) Assertion {
	return func(i int, t *testing.T, q *concurrent.ConcurrentDeque[any]) {
		if actual := q.PushBack(values...); actual != expected {
			t.Errorf("i: %d, PushBack() = %v; Expected: %v, values: %v", i, actual, expected, values)
		}
	}
}

func assertPushFront(expected int, values ...any) Assertion {
	return func(i int, t *testing.T, q *concurrent.ConcurrentDeque[any]) {
		if actual := q.PushFront(values...); actual != expected {
			t.Errorf("i: %d, PushFront() = %v; Expected: %v, values: %v", i, actual, expected, values)
		}
	}
}

func assertPopFront(n int, expected ...any) Assertion {
	return func(i int, t *testing.T, q *concurrent.ConcurrentDeque[any]) {
		if actual := q.PopFront(n); !slices.Equal(actual, expected) {
			t.Errorf("i: %d, PopFront() = %v; Expected: %v, n: %v", i, actual, expected, n)
		}
	}
}

func assertGetRange(startIndex int, stopIndex int, expected ...any) Assertion {
	return func(i int, t *testing.T, q *concurrent.ConcurrentDeque[any]) {
		if actual := q.GetRange(startIndex, stopIndex); !slices.Equal(actual, expected) {
			t.Errorf("i: %d, GetRange() = %v; Expected: %v, startIndex: %v, stopIndex: %v", i, actual, expected, startIndex, stopIndex)
		}
	}
}

func assertPopFrontAsync(timeout time.Duration, expected ...any) Assertion {
	return func(i int, t *testing.T, q *concurrent.ConcurrentDeque[any]) {
		var wg sync.WaitGroup

		ctx, cancel := context.WithCancel(context.Background())
		time.AfterFunc(timeout, func() {
			cancel()
		})

		for i, e := range expected {
			c := q.PopFrontAsync(timeout)
			wg.Go(func() {
				select {
				case <-ctx.Done():
					t.Errorf("i: %d, Cancelled PopFrontAsync due to timeout! Expected: %v, timeout: %v", i, e, timeout)
				case actual := <-c:
					if len(actual) != 1 || actual[0] != e {
						t.Errorf("i: %d, PopFrontAsync() = %v; Expected: %v, timeout: %v", i, actual, e, timeout)
					}
				}
			})
		}

		q.PushBack(expected...)
		wg.Wait()
	}
}

func TestConcurrentDeque(t *testing.T) {
	var TestCases = []TestCase{
		{
			name: "0 size operations",
			steps: []Assertion{
				assertLen(0),
				assertGetRange(0, 0),
				assertPopFront(1),
				assertPopFront(-1),
			},
		},
		{
			name: "Basic pushing and popping",
			steps: []Assertion{
				assertPushBack(1, "foo"),
				assertLen(1),
				assertGetRange(0, -1, "foo"),

				assertPushFront(2, "bar"),
				assertLen(2),
				assertGetRange(0, -1, "bar", "foo"),

				assertPushFront(4, "fez", "baz"),
				assertLen(4),
				assertGetRange(0, -1, "baz", "fez", "bar", "foo"),

				assertPopFront(3, "baz", "fez", "bar"),
				assertLen(1),
				assertGetRange(0, -1, "foo"),

				assertPushFront(5, "bar", "1", "2", "3"),
				assertLen(5),
				assertGetRange(0, -1, "3", "2", "1", "bar", "foo"),
			},
		},
		{
			name: "Blocking pop",
			steps: []Assertion{
				assertPopFrontAsync(10*time.Millisecond, "1", "2", "3", "4", "5", "6", "7"),
			},
		},
	}

	for _, tt := range TestCases {
		t.Run(tt.name, func(t *testing.T) {
			q := concurrent.NewConcurrentDeque[any]()
			for i, step := range tt.steps {
				step(i, t, q)
			}
		})
	}
}
