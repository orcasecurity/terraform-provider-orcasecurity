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
	"terraform-provider-orcasecurity/orcasecurity/internal/backoff"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// After a 504/timeout the backend may still be saving, so the lookup is retried (about 30s total).
var pollDelays = []time.Duration{0, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}

// A failed snapshot returns nil, which disables adoption.
func snapshot(ctx context.Context, what string, list api_client.IDLister) map[string]bool {
	ids, err := list()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Could not list existing %s before create; an ambiguous create will not be recovered: %v", what, err))
		return nil
	}
	return ids
}

func recoverID(ctx context.Context, createErr error, what string, before map[string]bool, list api_client.IDLister) (id, warning string, err error) {
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
	before := snapshot(ctx, op.What, op.IDs)
	created, err := op.Create()
	if err == nil {
		if created == nil || op.ID(created) == "" {
			return nil, fmt.Errorf("the create request for %s succeeded but returned no object id; check the Orca UI and import it if it was created", op.What)
		}
		return created, nil
	}
	id, warning, err := recoverID(ctx, err, op.What, before, op.IDs)
	if err != nil {
		return nil, err
	}
	created, err = op.Get(id)
	if err != nil || created == nil {
		return nil, api_client.CreatedButUnreadableError(op.What, id, err)
	}
	diags.AddWarning("Recovered "+op.What+" after an ambiguous create", warning)
	return created, nil
}

func newSince(ctx context.Context, before map[string]bool, list api_client.IDLister) ([]string, time.Duration, error) {
	var created []string
	waited, err := backoff.Poll(ctx, pollDelays, func() (bool, error) {
		after, err := list()
		if err != nil {
			return false, err
		}
		created = created[:0]
		for id := range after {
			if !before[id] {
				created = append(created, id)
			}
		}
		return len(created) > 0, nil
	})
	if err != nil || len(created) == 0 {
		return nil, waited, err
	}
	sort.Strings(created)
	return created, waited, nil
}
