package shift_left_repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/backoff"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

var moveRetryDelays = []time.Duration{0, time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second,
	15 * time.Second, 15 * time.Second, 15 * time.Second}

func moveRepository(ctx context.Context, ops repoOps, ctxID, targetProjectID string) error {
	settled := false
	waited, err := backoff.Poll(ctx, moveRetryDelays, func() (bool, error) {
		done, err := settleMove(ops, ops.client.MoveRepositoryContexts(targetProjectID, []string{ctxID}), targetProjectID)
		settled = done
		return done, err
	})
	if settled || err == nil || ctx.Err() != nil {
		return err
	}
	return fmt.Errorf("repository context %s is still not in project %s after retrying for %s: %w",
		ctxID, targetProjectID, waited, err)
}

// The move's outcome is settled by the repository's project on a re-read, never by a refusal alone.
func settleMove(ops repoOps, moveErr error, targetProjectID string) (done bool, err error) {
	if moveErr == nil {
		return true, nil
	}
	if !errors.Is(moveErr, api_client.ErrAmbiguousWrite) && !errors.Is(moveErr, api_client.ErrMoveInProgress) &&
		!errors.Is(moveErr, api_client.ErrMoveAlreadyInTarget) {
		return true, moveErr
	}
	row, findErr := ops.find()
	switch {
	case findErr != nil:
		return true, fmt.Errorf("%w; reading the repository back to confirm the move also failed: %w", moveErr, findErr)
	case row == nil:
		return true, fmt.Errorf("%w; the repository was not found when reading it back to confirm the move", moveErr)
	case row.ProjectID == targetProjectID:
		return true, nil
	case errors.Is(moveErr, api_client.ErrMoveAlreadyInTarget):
		return true, fmt.Errorf("%w, but reading it back shows project %s", moveErr, row.ProjectID)
	}
	return false, moveErr
}

// Move then PATCH are independent; on PATCH failure move the repo back so live matches state.
func rollbackProjectMove(ctx context.Context, ops repoOps, state *RepoConfigFields, movedTo, ctxID string, diags *diag.Diagnostics) {
	priorProjectID := state.ProjectID.ValueString()
	const detail = "The repository was moved to the configured project but applying its configuration failed"
	if !tfconv.Known(state.ProjectID) || priorProjectID == "" {
		diags.AddWarning(
			fmt.Sprintf("%s repository left in its new project", ops.traits.name),
			fmt.Sprintf("%s, and state does not record which project it came from, so the move could not be "+
				"undone. The repository is in project %s; the next apply continues from there.", detail, movedTo),
		)
		return
	}
	if err := moveRepository(ctx, ops, ctxID, priorProjectID); err != nil {
		diags.AddWarning(
			fmt.Sprintf("%s repository left in project %s", ops.traits.name, movedTo),
			fmt.Sprintf("%s, and moving it back to project %s did not succeed: %s.\n\n"+
				"The repository stays in project %s. Re-running terraform apply continues from there and "+
				"retries the configuration; to restore the previous placement instead, move the repository "+
				"to project %s in the Orca console once its current move has finished.",
				detail, priorProjectID, err.Error(), movedTo, priorProjectID),
		)
	}
}
