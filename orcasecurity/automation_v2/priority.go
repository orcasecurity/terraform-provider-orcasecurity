package automation_v2

// Priority is a single global evaluation order shared by every automation in
// the organization, written through a dedicated endpoint rather than the
// automation CRUD payload. This file holds everything the resource needs to
// manage it: the write path used by Create/Update and the refresh logic used
// by Read.
//
// Backend semantics (base_api automations set_priority):
//   - The server clamps the requested value to the organization's current
//     highest priority (Least(new, Max(priority))), NOT to the automation
//     count — legacy data can have gaps and duplicates.
//   - A full CRUD update (PUT /api/automations/{id}) resets every action's
//     status to ACTIVE as a side effect, so priority-only changes must go
//     through the priority endpoint alone, like the UI does.

import (
	"context"
	"fmt"
	"reflect"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// applyPriority moves the automation to the requested evaluation-order
// position and returns the priority the server actually assigned. The server
// silently clamps values above the organization's current highest priority,
// so callers must compare the returned value with the requested one and
// surface a diagnostic on mismatch.
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
		"priority %d exceeds the organization's current highest priority; the server placed the automation at priority %d. "+
			"Terraform recorded the actual value — align priority in the configuration and re-apply.",
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

// modelsEqualIgnoringPriority reports whether two resource models are
// identical apart from Priority. Update uses it to detect priority-only
// changes, which must skip the full CRUD update (the backend resets every
// action's status to ACTIVE on CRUD updates).
func modelsEqualIgnoringPriority(a, b automationV2ResourceModel) bool {
	a.Priority = types.Int64Null()
	b.Priority = types.Int64Null()
	return reflect.DeepEqual(a, b)
}

// applyPlanPriorityOnCreate sets the requested priority on a just-created
// automation, if any was requested. Failures surface as warnings, never
// errors: a Create error would mark the freshly created automation as tainted
// and force a destroy/recreate on the next apply, when the automation itself
// is perfectly healthy. Instead the state records what the server actually
// holds, so the next plan shows a priority diff and the next apply retries
// through the priority-only update path.
func (r *automationV2Resource) applyPlanPriorityOnCreate(plan *automationV2ResourceModel, instanceID string, resp *resource.CreateResponse) {
	if plan.Priority.IsNull() || plan.Priority.IsUnknown() {
		return
	}
	requested := plan.Priority.ValueInt64()
	actual, err := r.applyPriority(instanceID, requested)
	if err != nil {
		plan.Priority = types.Int64Null()
		resp.Diagnostics.AddWarning(
			"Automation V2 created, but priority was not set",
			fmt.Sprintf("Automation %s was created, but setting priority failed: %s. The next apply retries setting the priority.", instanceID, err.Error()),
		)
		return
	}
	if actual != requested {
		plan.Priority = types.Int64Value(actual)
		resp.Diagnostics.AddWarning("Automation V2 priority clamped by the server", clampErrorDetail(requested, actual))
	}
}

// updatePriorityOnly handles an Update whose only change is priority: it
// talks solely to the priority endpoint, mirroring the UI. Skipping the CRUD
// update avoids its side effect of resetting all action statuses to ACTIVE.
func (r *automationV2Resource) updatePriorityOnly(ctx context.Context, plan *automationV2ResourceModel, resp *resource.UpdateResponse) {
	instance, err := r.apiClient.GetAutomationV2(plan.ID.ValueString())
	if err == nil && instance == nil {
		err = fmt.Errorf("received nil instance from API")
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Automation V2",
			fmt.Sprintf("Could not read Automation V2 ID %s: %s", plan.ID.ValueString(), err.Error()),
		)
		return
	}
	resp.Diagnostics.Append(r.resolvePlanPriorityOnUpdate(plan, instance)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// resolvePlanPriorityOnUpdate PUTs the requested priority only when it differs
// from the automation's current server-side priority, and does nothing when
// the plan's priority is null (user stopped tracking it: no API call, no
// error). It mutates plan.Priority to the value the server actually holds and
// returns diagnostics for any failure or clamp; it never writes state. The
// caller commits plan to state exactly once, so state stays consistent whether
// this succeeds, clamps, or errors (an error still persists the server's real
// priority rather than hiding the remote mutation).
func (r *automationV2Resource) resolvePlanPriorityOnUpdate(plan *automationV2ResourceModel, verifyInstance *api_client.AutomationV2) diag.Diagnostics {
	var diags diag.Diagnostics
	if plan.Priority.IsNull() || plan.Priority.IsUnknown() {
		return diags
	}
	requested := plan.Priority.ValueInt64()
	if verifyInstance.Priority != nil && *verifyInstance.Priority == requested {
		return diags
	}
	actual, err := r.applyPriority(plan.ID.ValueString(), requested)
	if err != nil {
		plan.Priority = types.Int64PointerValue(verifyInstance.Priority)
		diags.AddError(
			"Error setting Automation V2 priority",
			fmt.Sprintf("Could not set priority for Automation V2 ID %s: %s", plan.ID.ValueString(), err.Error()),
		)
		return diags
	}
	if actual != requested {
		plan.Priority = types.Int64Value(actual)
		diags.AddError("Automation V2 priority out of range", clampErrorDetail(requested, actual))
	}
	return diags
}
