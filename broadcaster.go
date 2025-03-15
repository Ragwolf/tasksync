package broadcaster

import (
	"context"
	"sync"
)

type TaskBroadcaster[T any] struct {
	mu        sync.Mutex
	listeners map[chan Result[T]]struct{}

	running bool
}

type Result[T any] struct {
	Value T
	Err   error
}

type TaskRunner[T any] func(context.Context) (T, error)

// New creates a new TaskBroadcaster instance.
func New[T any]() *TaskBroadcaster[T] {
	return &TaskBroadcaster[T]{
		listeners: make(map[chan Result[T]]struct{}),
	}
}

func (b *TaskBroadcaster[T]) Listen(ctx context.Context, runner TaskRunner[T]) (T, error) {
	ch := make(chan Result[T], 1)

	b.mu.Lock()
	b.listeners[ch] = struct{}{}

	// Task is already running, wait for the result
	if b.running {
		b.mu.Unlock()

		return b.waitForResult(ctx, ch)
	}

	b.running = true
	b.mu.Unlock()

	go b.runTask(ctx, runner)

	return b.waitForResult(ctx, ch)
}

func (b *TaskBroadcaster[T]) runTask(ctx context.Context, runner TaskRunner[T]) {
	// Run the task
	result, err := runner(ctx)

	b.mu.Lock()
	defer b.mu.Unlock()

	// Broadcast the results
	for listener := range b.listeners {
		listener <- Result[T]{Value: result, Err: err}
		close(listener)
	}

	// Clean up
	b.listeners = make(map[chan Result[T]]struct{})
	b.running = false
}

func (b *TaskBroadcaster[T]) waitForResult(ctx context.Context, ch chan Result[T]) (T, error) {
	select {
	case result := <-ch:
		return result.Value, result.Err
	case <-ctx.Done():
		b.mu.Lock()
		delete(b.listeners, ch) // Clean up the listener
		b.mu.Unlock()
		return *new(T), ctx.Err()
	}
}
