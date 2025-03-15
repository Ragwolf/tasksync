package broadcaster_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ragwolf/broadcaster"
)

func TestBroadcaster(t *testing.T) {
	b := broadcaster.New[string]()

	timeout := time.Duration(10) * time.Second

	var wg sync.WaitGroup

	for i := 1; i < 10; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
			defer cancelFunc()

			value, err := b.Listen(ctx, longRunningTask(i))
			if err != nil {
				t.Error(err)
			}
			t.Logf("Go Func %d: %v", i, value)
		}(i)
	}

	wg.Wait()
}

func TestBroadcaster_CancelContext(t *testing.T) {
	b := broadcaster.New[string]()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel the context immediately

	value, err := b.Listen(ctx, func(ctx context.Context) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(5 * time.Second):
			return "should not happen", nil
		}
	})

	if err == nil || err.Error() != "context canceled" {
		t.Errorf("expected context canceled error, got: %v", err)
	}
	if value != "" {
		t.Errorf("expected empty value, got: %v", value)
	}
}

func longRunningTask(value int) broadcaster.TaskRunner[string] {
	return func(ctx context.Context) (string, error) {
		fmt.Printf("starting longRunningTask: %d\n", value)

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Duration(5) * time.Second):
			return fmt.Sprintf("longRunningTask: %d", value), nil
		}
	}
}
