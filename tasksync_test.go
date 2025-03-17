package tasksync

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestTaskSync_SingleTask tests that a single task runs and broadcasts its result to multiple listeners.
func TestTaskSync_SingleTask(t *testing.T) {
	ts := NewTaskSync[string]()

	runner := func(ctx context.Context) (string, error) {
		return "Hello, World!", nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			result, err := ts.Run(context.Background(), runner)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != "Hello, World!" {
				t.Errorf("Expected 'Hello, World!', got '%s'", result)
			}
		}()
	}

	wg.Wait()
}

// TestTaskSync_Error tests that a task returning an error broadcasts the error to all listeners.
func TestTaskSync_Error(t *testing.T) {
	ts := NewTaskSync[string]()

	expectedErr := errors.New("task failed")
	runner := func(ctx context.Context) (string, error) {
		return "", expectedErr
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := ts.Run(context.Background(), runner)
			if err != expectedErr {
				t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
			}
		}()
	}

	wg.Wait()
}

// TestTaskSync_ContextCanceled tests that a canceled context results in an error for all listeners.
func TestTaskSync_ContextCanceled(t *testing.T) {
	ts := NewTaskSync[string]()

	runner := func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "Hello, World!", nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel the context immediately

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := ts.Run(ctx, runner)
			if err != context.Canceled {
				t.Errorf("Expected context.Canceled, got '%v'", err)
			}
		}()
	}

	wg.Wait()
}

// TestTaskSync_ConcurrentRuners tests that multiple concurrent listeners receive the same result.
func TestTaskSync_ConcurrentRuners(t *testing.T) {
	ts := NewTaskSync[string]()

	runner := func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "Hello, World!", nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			result, err := ts.Run(context.Background(), runner)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != "Hello, World!" {
				t.Errorf("Expected 'Hello, World!', got '%s'", result)
			}
		}()
	}

	wg.Wait()
}

// TestTaskSync_Cleanup tests that resources are cleaned up after the task completes.
func TestTaskSync_Cleanup(t *testing.T) {
	ts := NewTaskSync[string]()

	runner := func(ctx context.Context) (string, error) {
		return "Hello, World!", nil
	}

	_, err := ts.Run(context.Background(), runner)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !ts.isInactive() {
		t.Error("TaskSync should be inactive after task completion")
	}
}

// TestTaskSync_Panic tests that a panicking task does not crash the application.
func TestTaskSync_Panic(t *testing.T) {
	ts := NewTaskSync[string]()

	runner := func(ctx context.Context) (string, error) {
		panic("task panicked")
	}

	_, err := ts.Run(context.Background(), runner)
	if err == nil {
		t.Error("Expected an error, got nil")
	} else if err.Error() != "task panicked: task panicked" {
		t.Errorf("Expected 'task panicked: task panicked', got '%v'", err)
	}
}

// TestGroupedTaskSync_SingleKey tests that tasks are grouped by key and only one task runs per key.
func TestGroupedTaskSync_SingleKey(t *testing.T) {
	gts := NewGroupedTaskSync[string, string]()

	runner := func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "Hello, World!", nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			result, err := gts.Run(context.Background(), "key1", runner)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != "Hello, World!" {
				t.Errorf("Expected 'Hello, World!', got '%s'", result)
			}
		}()
	}

	wg.Wait()
}

// TestGroupedTaskSync_MultipleKeys tests that tasks for different keys run independently.
func TestGroupedTaskSync_MultipleKeys(t *testing.T) {
	gts := NewGroupedTaskSync[string, string]()

	runner := func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "Hello, World!", nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			key := string(rune('A' + i)) // Use different keys for each goroutine
			result, err := gts.Run(context.Background(), key, runner)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != "Hello, World!" {
				t.Errorf("Expected 'Hello, World!', got '%s'", result)
			}
		}(i)
	}

	wg.Wait()
}

// TestGroupedTaskSync_Cleanup tests that groups are cleaned up after tasks complete.
func TestGroupedTaskSync_Cleanup(t *testing.T) {
	gts := NewGroupedTaskSync[string, string]()

	runner := func(ctx context.Context) (string, error) {
		return "Hello, World!", nil
	}

	_, err := gts.Run(context.Background(), "key1", runner)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	gts.mu.Lock()
	defer gts.mu.Unlock()

	if _, exists := gts.groups["key1"]; exists {
		t.Error("Group should be cleaned up after task completion")
	}
}
