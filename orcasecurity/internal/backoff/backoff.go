// Package backoff holds the one context-aware sleep and poll loop shared by the API client and
// create recovery, plus the single test seam that makes them instant.
package backoff

import (
	"context"
	"sync"
	"time"
)

var sleep = func(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Sleep waits d, or returns ctx.Err() if ctx ends first.
func Sleep(ctx context.Context, d time.Duration) error {
	return sleep(ctx, d)
}

// Poll sleeps each delay in turn and then calls try, until try reports done. An error with done=false
// is retried; if no try is done, Poll returns the last try's error. waited is the total delay slept.
func Poll(ctx context.Context, delays []time.Duration, try func() (done bool, err error)) (waited time.Duration, err error) {
	var lastErr error
	for _, d := range delays {
		if err := Sleep(ctx, d); err != nil {
			return waited, err
		}
		waited += d
		done, err := try()
		if done {
			return waited, err
		}
		lastErr = err
	}
	return waited, lastErr
}

// FastForTest makes Sleep return at once for the rest of the test and records each requested delay.
// Tests that call it must not use t.Parallel.
func FastForTest(t interface{ Cleanup(func()) }) func() []time.Duration {
	var mu sync.Mutex
	var slept []time.Duration
	orig := sleep
	sleep = func(ctx context.Context, d time.Duration) error {
		mu.Lock()
		defer mu.Unlock()
		slept = append(slept, d)
		return ctx.Err()
	}
	t.Cleanup(func() { sleep = orig })
	return func() []time.Duration {
		mu.Lock()
		defer mu.Unlock()
		return append([]time.Duration(nil), slept...)
	}
}
