package backoff

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPoll_StopsWhenDone(t *testing.T) {
	slept := FastForTest(t)
	calls := 0
	waited, err := Poll(context.Background(), []time.Duration{0, time.Second, 2 * time.Second, 4 * time.Second}, func() (bool, error) {
		calls++
		return calls == 2, nil
	})
	if err != nil || calls != 2 || waited != time.Second || len(slept()) != 2 {
		t.Fatalf("err=%v calls=%d waited=%s slept=%v", err, calls, waited, slept())
	}
}

func TestPoll_ReturnsTheLastErrorWhenNeverDone(t *testing.T) {
	FastForTest(t)
	errs := []error{errors.New("first"), nil, errors.New("last")}
	calls := 0
	_, err := Poll(context.Background(), []time.Duration{0, 0, 0}, func() (bool, error) {
		calls++
		return false, errs[calls-1]
	})
	if err == nil || err.Error() != "last" || calls != 3 {
		t.Fatalf("err=%v calls=%d, want the last error after 3 tries", err, calls)
	}
}

func TestPoll_DoneWithAnErrorStops(t *testing.T) {
	FastForTest(t)
	calls := 0
	_, err := Poll(context.Background(), []time.Duration{0, 0, 0}, func() (bool, error) {
		calls++
		return true, errors.New("terminal")
	})
	if err == nil || calls != 1 {
		t.Fatalf("err=%v calls=%d, want the terminal error after 1 try", err, calls)
	}
}

func TestSleep_EndsWithTheContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Sleep(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
