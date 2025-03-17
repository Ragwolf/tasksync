package tasksync_test

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ragwolf/tasksync"
)

// ExampleTaskSync demonstrates how TaskSync synchronizes a task
// and broadcasts its result to multiple concurrent listeners.
func ExampleTaskSync() {
	// Create a new TaskSync instance
	ts := tasksync.NewTaskSync[string]()

	// Define a task runner
	runner := func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond) // Simulate a long-running task
		return "Hello, World!", nil
	}

	// Use TaskSync to run the task with multiple concurrent listeners
	ctx := context.Background()
	var wg sync.WaitGroup

	// Use a slice to collect results in order
	results := make([]string, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			result, err := ts.Run(ctx, runner)
			if err != nil {
				log.Fatalf("Task failed for goroutine %d: %v", i, err)
			}

			results[i] = fmt.Sprintf("Goroutine %d: %s\n", i, result)
		}(i)
	}

	wg.Wait()

	// Print the results in order
	for _, result := range results {
		fmt.Print(result)
	}

	// Output:
	// Goroutine 0: Hello, World!
	// Goroutine 1: Hello, World!
	// Goroutine 2: Hello, World!
	// Goroutine 3: Hello, World!
	// Goroutine 4: Hello, World!
}

// ExampleGroupedTaskSync demonstrates how GroupedTaskSync groups tasks by a key
// and ensures that only one task runs per key, even with multiple concurrent requests.
func ExampleGroupedTaskSync() {
	// Create a new GroupedTaskSync instance
	gts := tasksync.NewGroupedTaskSync[string, string]()

	// Define a task runner
	runner := func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond) // Simulate a long-running task
		return "Task completed!", nil
	}

	// Use GroupedTaskSync to run tasks for the same key with multiple concurrent listeners
	ctx := context.Background()
	key := "example-key"
	var wg sync.WaitGroup

	// Use a slice to collect results in order
	results := make([]string, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			result, err := gts.Run(ctx, key, runner)
			if err != nil {
				log.Fatalf("Task failed for goroutine %d: %v", i, err)
			}

			results[i] = fmt.Sprintf("Goroutine %d: %s\n", i, result)
		}(i)
	}

	wg.Wait()

	// Print the results in order
	for _, result := range results {
		fmt.Print(result)
	}

	// Output:
	// Goroutine 0: Task completed!
	// Goroutine 1: Task completed!
	// Goroutine 2: Task completed!
	// Goroutine 3: Task completed!
	// Goroutine 4: Task completed!
}
