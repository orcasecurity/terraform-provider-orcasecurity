// Shared compliance_frameworks reconciliation for custom alert resources.
package alert_common

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ReplaceFrameworks writes an alert, clearing its compliance framework links
// first when the write both replaces existing links and sets new ones: the API
// merges posted frameworks instead of replacing them.
//
// clearedButFailed reports that the clear landed remotely but the write after it
// did not, so the caller must persist empty frameworks.
func ReplaceFrameworks[T any](state, plan types.List, request T, clearLinks func(*T), write func(T) error) (clearedButFailed bool, err error) {
	if FrameworksCount(state) > 0 && FrameworksCount(plan) > 0 {
		cleared := request
		clearLinks(&cleared)
		if err := write(cleared); err != nil {
			return false, fmt.Errorf("could not clear the existing compliance frameworks: %w", err)
		}
		if err := write(request); err != nil {
			return true, fmt.Errorf("could not update alert, unexpected error: %w", err)
		}
		return false, nil
	}
	if err := write(request); err != nil {
		return false, fmt.Errorf("could not update alert, unexpected error: %w", err)
	}
	return false, nil
}

// FrameworksAfterFailedReplace reports the value state must hold when the clear
// landed remotely but the follow-up write did not: the links are gone on the
// backend, so state has to say so even though the apply failed.
func FrameworksAfterFailedReplace(clearedButFailed bool) (types.List, bool) {
	if !clearedButFailed {
		return types.ListNull(FrameworkObjectType()), false
	}
	return types.ListValueMust(FrameworkObjectType(), nil), true
}

// ReportFrameworkWriteFailure records a failed alert update. When the clear
// landed remotely the links are already gone, so state has to say so before the
// error surfaces — otherwise the next plan is built on links that no longer
// exist.
func ReportFrameworkWriteFailure[T any](ctx context.Context, resp *resource.UpdateResponse, plan *T, frameworks *types.List, clearedButFailed bool, err error) {
	if cleared, persist := FrameworksAfterFailedReplace(clearedButFailed); persist {
		*frameworks = cleared
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	}
	resp.Diagnostics.AddError("Error updating Alert", err.Error())
}
