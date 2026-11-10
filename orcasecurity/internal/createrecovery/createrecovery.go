// Package createrecovery recovers the id of a create whose POST failed with an ambiguous outcome
// (api_client.ErrAmbiguousWrite): the server may have committed it, and the client never replays it.
package createrecovery

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// After a 504/timeout the backend may still be saving, so the lookup is retried (about 30s total).
// FastForTest swaps these package vars, so tests that call it must not use t.Parallel.
var (
	PollDelays = []time.Duration{0, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}
	Sleep      = sleepCtx
)

// Lister returns the ids of every existing object that matches what is being created. Names are not
// unique, so recovery diffs id sets rather than trusting a name match alone.
type Lister func() (map[string]bool, error)

// Snapshot takes the pre-create id set. A failed snapshot returns nil, which disables adoption.
func Snapshot(ctx context.Context, what string, list Lister) map[string]bool {
	ids, err := list()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Could not list existing %s before create; an ambiguous create will not be recovered: %v", what, err))
		return nil
	}
	return ids
}

// SameName compares names the way the server may store them (surrounding whitespace trimmed).
// A looser match is safe: adoption still requires exactly one id absent from the snapshot.
func SameName(a, b string) bool {
	return strings.TrimSpace(a) == strings.TrimSpace(b)
}

// Recover returns the created id when createErr is ambiguous and exactly one new match appeared.
// what describes the object for messages, e.g. `iac policy named "x"`. warning is set on adoption.
func Recover(ctx context.Context, createErr error, what string, before map[string]bool, list Lister) (id, warning string, err error) {
	if !errors.Is(createErr, api_client.ErrAmbiguousWrite) {
		return "", "", createErr
	}
	if before == nil {
		return "", "", fmt.Errorf("%w; look for a %s and import it before re-applying", createErr, what)
	}
	created, waited, listErr := newSince(ctx, before, list)
	if listErr != nil {
		return "", "", fmt.Errorf("%w; checking whether it was created also failed (%v) — look for a %s and import it before re-applying",
			createErr, listErr, what)
	}
	switch len(created) {
	case 0:
		return "", "", fmt.Errorf("%w; no new %s appeared after %s — check the Orca UI before re-applying, and import it if it shows up",
			createErr, what, waited)
	case 1:
		return created[0], fmt.Sprintf("The create request for %s failed without a definite answer (%v), but it was created as %s; it is now tracked in state.",
			what, createErr, created[0]), nil
	default:
		return "", "", fmt.Errorf("%w; %d new matches for %s appeared (%s) — import the intended one and delete the rest",
			createErr, len(created), what, strings.Join(created, ", "))
	}
}

// Run executes op. On an ambiguous POST it adopts the single new matching object, reads it back and
// adds a warning to diags; any other outcome is returned as an error.
func Run[T any](ctx context.Context, op api_client.CreateOp[T], diags *diag.Diagnostics) (*T, error) {
	before := Snapshot(ctx, op.What, op.IDs)
	created, err := op.Create()
	if err == nil {
		return created, nil
	}
	id, warning, err := Recover(ctx, err, op.What, before, op.IDs)
	if err != nil {
		return nil, err
	}
	created, err = op.Get(id)
	if err == nil && created == nil {
		err = errors.New("not found")
	}
	if err != nil {
		return nil, fmt.Errorf("%s was created as %s after an ambiguous create, but reading it back failed (%v); import it before re-applying", op.What, id, err)
	}
	diags.AddWarning("Recovered "+op.What+" after an ambiguous create", warning)
	return created, nil
}

func newSince(ctx context.Context, before map[string]bool, list Lister) ([]string, time.Duration, error) {
	var waited time.Duration
	var lastErr error
	for _, d := range PollDelays {
		if err := Sleep(ctx, d); err != nil {
			return nil, waited, err
		}
		waited += d
		after, err := list()
		if err != nil {
			lastErr = err
			continue
		}
		lastErr = nil
		var created []string
		for id := range after {
			if !before[id] {
				created = append(created, id)
			}
		}
		if len(created) > 0 {
			sort.Strings(created)
			return created, waited, nil
		}
	}
	return nil, waited, lastErr
}

func sleepCtx(ctx context.Context, d time.Duration) error {
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

// FastForTest removes the backoff between lookups for the duration of a test.
func FastForTest(t interface{ Cleanup(func()) }) {
	delays, sleep := PollDelays, Sleep
	PollDelays = []time.Duration{0, time.Millisecond, time.Millisecond, time.Millisecond}
	Sleep = func(context.Context, time.Duration) error { return nil }
	t.Cleanup(func() { PollDelays, Sleep = delays, sleep })
}
