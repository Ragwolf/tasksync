package tasksync

import (
	"context"
	"fmt"
	"sync"
)

// TaskSync synchronizes the execution of a task and broadcasts its result to all listeners.
// It ensures that only one instance of the task runs at a time, and all concurrent calls
// for the same task wait for the result instead of executing the task again.
type TaskSync[T any] struct {
	mu        sync.Mutex                  // Protects access to listeners and running state
	listeners map[chan result[T]]struct{} // Map of channels to broadcast results to
	running   bool                        // Indicates whether the task is currently running
}

// TaskRunner defines a function that executes a task and returns a result or an error.
// The function should respect the provided context for cancellation and timeouts.
type TaskRunner[T any] func(context.Context) (T, error)

// result holds the output of a task, including the value and any error that occurred.
type result[T any] struct {
	Value T     // The result value returned by the task
	Err   error // Any error that occurred during task execution
}

// NewTaskSync creates and returns a new TaskSync instance.
// The instance is ready to manage a single task and broadcast its result to listeners.
func NewTaskSync[T any]() *TaskSync[T] {
	return &TaskSync[T]{
		listeners: make(map[chan result[T]]struct{}),
	}
}

// Run executes the provided task runner and returns its result. If the task is already
// running, Run waits for the result instead of starting a new task. The context is used
// to control cancellation and timeouts.
//
// Returns:
//   - The result of the task if it completes successfully.
//   - An error if the context is canceled, the task fails, or a timeout occurs.
func (b *TaskSync[T]) Run(ctx context.Context, runner TaskRunner[T]) (T, error) {
	ch := make(chan result[T], 1) // Channel to receive the task result

	b.mu.Lock()
	b.listeners[ch] = struct{}{} // Register the listener

	// If the task is already running, wait for the result
	if b.running {
		b.mu.Unlock()

		return b.waitForResult(ctx, ch)
	}

	// Start the task
	b.running = true
	b.mu.Unlock()

	go b.runTask(ctx, runner) // Run the task in a goroutine

	return b.waitForResult(ctx, ch) // Wait for the result
}

// runTask executes the task runner, broadcasts the result to all listeners, and cleans up.
func (b *TaskSync[T]) runTask(ctx context.Context, runner TaskRunner[T]) {
	// Recover from any panics in the task runner
	defer func() {

		if r := recover(); r != nil {
			b.mu.Lock()
			// Broadcast the panic as an error to all listeners
			for listener := range b.listeners {
				listener <- result[T]{Err: fmt.Errorf("task panicked: %v", r)}
				close(listener)
			}
			b.mu.Unlock()
		}

		// Clean up
		b.mu.Lock()
		b.listeners = make(map[chan result[T]]struct{})
		b.running = false
		b.mu.Unlock()
	}()

	// Execute the task
	res, err := runner(ctx)

	b.mu.Lock()
	defer b.mu.Unlock()

	// Broadcast the result to all listeners
	for listener := range b.listeners {
		listener <- result[T]{Value: res, Err: err}
		close(listener) // Close the channel to signal completion
	}
}

// waitForResult waits for the task result on the provided channel or until the context is done.
// If the context is canceled, it cleans up the listener and returns an error.
func (b *TaskSync[T]) waitForResult(ctx context.Context, ch chan result[T]) (T, error) {
	select {
	case res := <-ch: // Task completed successfully
		return res.Value, res.Err
	case <-ctx.Done(): // Context canceled or timed out
		b.mu.Lock()
		delete(b.listeners, ch) // Clean up the listener
		b.mu.Unlock()
		return *new(T), ctx.Err()
	}
}

// isInactive checks whether the TaskSync is inactive, meaning no task is running
// and there are no registered listeners.
func (b *TaskSync[T]) isInactive() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return !b.running && len(b.listeners) == 0
}

// GroupedTaskSync groups tasks by a key and ensures that only one task runs per key.
// It uses TaskSync internally to manage tasks for each key.
type GroupedTaskSync[T any, K comparable] struct {
	mu     sync.Mutex
	groups map[K]*TaskSync[T]
}

// NewGroupedTaskSync creates and returns a new GroupedTaskSync instance.
// The instance is ready to manage tasks grouped by a key of type K.
func NewGroupedTaskSync[T any, K comparable]() *GroupedTaskSync[T, K] {
	return &GroupedTaskSync[T, K]{
		groups: map[K]*TaskSync[T]{},
	}
}

// Run executes the provided task runner for the given key. If a task is already
// running for the key, Run waits for the result instead of starting a new task.
// The context is used to control cancellation and timeouts.
//
// Returns:
//   - The result of the task if it completes successfully.
//   - An error if the context is canceled, the task fails, or a timeout occurs.
func (g *GroupedTaskSync[T, K]) Run(ctx context.Context, key K, runner TaskRunner[T]) (T, error) {
	g.mu.Lock()

	// Get or create a TaskSync instance for the key
	b, exists := g.groups[key]
	if !exists {
		b = NewTaskSync[T]()
		g.groups[key] = b
	}

	g.mu.Unlock()

	// Execute the task and wait for the result
	result, err := b.Run(ctx, runner)

	// Clean up the TaskSync instance if it's no longer active
	g.mu.Lock()
	if b.isInactive() {
		delete(g.groups, key)
	}
	g.mu.Unlock()

	return result, err
}
