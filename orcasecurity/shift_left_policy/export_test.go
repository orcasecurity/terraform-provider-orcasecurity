package shift_left_policy

import (
	"context"
	"time"
)

// FastAmbiguousCreatePoll removes the real backoff between post-create lookups for the test.
func FastAmbiguousCreatePoll(t interface{ Cleanup(func()) }) {
	delays, sleep := ambiguousCreatePollDelays, ambiguousCreateSleep
	ambiguousCreatePollDelays = []time.Duration{0, time.Millisecond, time.Millisecond, time.Millisecond}
	ambiguousCreateSleep = func(context.Context, time.Duration) error { return nil }
	t.Cleanup(func() { ambiguousCreatePollDelays, ambiguousCreateSleep = delays, sleep })
}
