package concurrent_test

import (
	"fmt"
	"time"

	"github.com/codecrafters-io/redis-starter-go/concurrent"
)

// Use your actual struct and methods here
func main() {
	m := concurrent.NewConcurrentMap[string, string]()
	key := "race-key"

	for i := range 100000 {
		// 1. Set a value that expires almost immediately
		m.Set(key, "old", 1*time.Nanosecond)

		// 2. Immediately overwrite it with a long-lived value
		m.Set(key, "new", 1*time.Hour)

		// 3. Small pause to let any runaway timers catch up
		time.Sleep(10 * time.Microsecond)

		// 4. CHECK: If the "old" timer fired and lacked the check,
		// it might have deleted the "new" value.
		if _, ok := m.Get(key); !ok {
			fmt.Printf("REPRODUCED! Key was deleted by an old timer on iteration %d\n", i)
			return
		}
	}
	fmt.Println("Could not reproduce. Your logic might be safe or the CPU was too fast.")
}
