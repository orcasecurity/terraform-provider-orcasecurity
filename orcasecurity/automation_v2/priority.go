package automation_v2

// Priority is a single global evaluation order shared by every automation in
// the organization, written through a dedicated endpoint rather than the
// automation CRUD payload. This file holds everything the resource needs to
// manage it: the write path used by Create/Update and the refresh logic used
// by Read.

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// applyPriority moves the automation to the requested evaluation-order
// position and returns the priority the server actually assigned. The server
// silently clamps values above the automation count, so callers must compare
// the returned value with the requested one and surface a diagnostic on
// mismatch.
func (r *automationV2Resource) applyPriority(id string, requested int64) (int64, error) {
	instance, err := r.apiClient.SetAutomationV2Priority(id, requested)
	if err != nil {
		return 0, err
	}
	if instance == nil || instance.Priority == nil {
		return 0, fmt.Errorf("priority endpoint returned no priority value")
	}
	return *instance.Priority, nil
}

// clampErrorDetail formats the diagnostic message shown to users when the
// server clamps a requested priority to a lower value than requested.
func clampErrorDetail(requested, actual int64) string {
	return fmt.Sprintf(
		"priority %d exceeds the number of automations; the server placed the automation at priority %d. "+
			"The automation is tracked in state — lower priority in the configuration and re-apply.",
		requested, actual)
}

// refreshPriority updates the model's priority from the API instance, but only
// when priority is already tracked (non-null) in state. Untracked priority
// stays null so configurations that never set it see no drift noise from
// external reordering.
func refreshPriority(state *automationV2ResourceModel, instance *api_client.AutomationV2) {
	if state.Priority.IsNull() {
		return
	}
	if instance == nil {
		state.Priority = types.Int64Null()
		return
	}
	state.Priority = types.Int64PointerValue(instance.Priority)
}

// applyPlanPriorityOnCreate sets the requested priority on a just-created
// automation, if any was requested. It writes state before returning true on
// error so the automation is tracked rather than orphaned. Split out of
// Create to keep that function's cognitive complexity low.
func (r *automationV2Resource) applyPlanPriorityOnCreate(ctx context.Context, plan *automationV2ResourceModel, instanceID string, resp *resource.CreateResponse) bool {
	if plan.Priority.IsNull() || plan.Priority.IsUnknown() {
		return false
	}
	requested := plan.Priority.ValueInt64()
	actual, err := r.applyPriority(instanceID, requested)
	if err != nil {
		plan.Priority = types.Int64Null()
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError(
			"Error setting Automation V2 priority",
			fmt.Sprintf("Automation %s was created, but setting priority failed: %s", instanceID, err.Error()),
		)
		return true
	}
	if actual != requested {
		plan.Priority = types.Int64Value(actual)
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("Automation V2 priority out of range", clampErrorDetail(requested, actual))
		return true
	}
	return false
}

// applyPlanPriorityOnUpdate PUTs the requested priority only when it differs
// from the automation's current server-side priority, and does nothing when
// the plan's priority is null (user stopped tracking it: no API call, no
// error). Split out of Update to keep that function's cognitive complexity
// low.
func (r *automationV2Resource) applyPlanPriorityOnUpdate(ctx context.Context, plan *automationV2ResourceModel, verifyInstance *api_client.AutomationV2, resp *resource.UpdateResponse) bool {
	if plan.Priority.IsNull() || plan.Priority.IsUnknown() {
		return false
	}
	requested := plan.Priority.ValueInt64()
	if verifyInstance.Priority != nil && *verifyInstance.Priority == requested {
		return false
	}
	actual, err := r.applyPriority(plan.ID.ValueString(), requested)
	if err != nil {
		// Unlike Create, no explicit State.Set here: on an errored Update the
		// framework keeps the prior state, which is exactly what we want (the
		// automation already exists and its old priority is still accurate).
		resp.Diagnostics.AddError(
			"Error setting Automation V2 priority",
			fmt.Sprintf("Could not set priority for Automation V2 ID %s: %s", plan.ID.ValueString(), err.Error()),
		)
		return true
	}
	if actual != requested {
		plan.Priority = types.Int64Value(actual)
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("Automation V2 priority out of range", clampErrorDetail(requested, actual))
		return true
	}
	return false
}
