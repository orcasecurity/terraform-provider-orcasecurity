package shift_left_policy

import (
	"context"
	"fmt"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                   = &shiftLeftPolicyResource{}
	_ resource.ResourceWithConfigure      = &shiftLeftPolicyResource{}
	_ resource.ResourceWithImportState    = &shiftLeftPolicyResource{}
	_ resource.ResourceWithValidateConfig = &shiftLeftPolicyResource{}
)

type shiftLeftPolicyResource struct {
	apiClient *api_client.APIClient
}

func NewShiftLeftPolicyResource() resource.Resource {
	return &shiftLeftPolicyResource{}
}

func (r *shiftLeftPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_policy"
}

func (r *shiftLeftPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *shiftLeftPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides an AppSec (Shift Left) policy resource. Use this resource to create and manage AppSec scan policies in Orca Security.",
		Attributes:  resourceSchemaAttributes(),
		Blocks:      resourceSchemaBlocks(),
	}
}

// ValidateConfig rejects combinations the API cannot apply. scm_posture policies
// are scoped via scm_posture.scope (installation/unit IDs); there is no
// /api/shiftleft/scm_posture/policies/{id}/projects/ endpoint, so setting
// projects_ids would 404 after the policy body had already been written.
func (r *shiftLeftPolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config shiftLeftPolicyResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.Type.IsUnknown() || config.ProjectsIds.IsUnknown() {
		return
	}
	if config.Type.ValueString() == "scm_posture" && !config.ProjectsIds.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("projects_ids"),
			"Unsupported attribute for scm_posture",
			"scm_posture policies cannot use projects_ids — the API has no project-attachment endpoint for this type. "+
				"Scope them with scm_posture.scope (installation/unit IDs) instead.",
		)
	}
}

func parseImportID(id string) (policyType, policyID string, err error) {
	policyType, policyID, ok := strings.Cut(id, "/")
	if !ok || policyType == "" || policyID == "" {
		return "", "", fmt.Errorf("import ID must be in the format <type>/<id>, got %q", id)
	}
	return policyType, policyID, nil
}

func (r *shiftLeftPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	policyType, policyID, err := parseImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	instance, err := r.apiClient.GetShiftLeftPolicy(policyType, policyID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing AppSec policy", err.Error())
		return
	}
	if instance == nil {
		resp.Diagnostics.AddError("Error importing AppSec policy", fmt.Sprintf("Policy %s/%s not found.", policyType, policyID))
		return
	}
	// The org-wide built-in scm_posture policy has a dedicated singleton resource with
	// its own PUT endpoint and locked fields. Owning it from two resource types at once
	// invites conflicting writes, so this resource refuses to take it.
	if policyType == "scm_posture" && instance.Builtin {
		resp.Diagnostics.AddError(
			"Built-in scm_posture policy has a dedicated resource",
			"The org-wide built-in SCM posture policy is managed exclusively by "+
				"orcasecurity_shift_left_scm_posture_default_policy, which adopts it on apply without "+
				"an import. Use orcasecurity_shift_left_policy only for custom scm_posture policies.",
		)
		return
	}

	state := apiToState(instance, nil)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), state.ID.ValueString())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), policyType)...)
}

func (r *shiftLeftPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan shiftLeftPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiPolicy, diags := planToAPI(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyType := plan.Type.ValueString()
	// Projects sync via dedicated endpoint — main body omitempty drops empty projects_ids.
	apiPolicy.ProjectsIds = nil
	if !r.applyCatalog(&plan, &apiPolicy, &resp.Diagnostics) {
		return
	}

	instance, err := r.apiClient.CreateShiftLeftPolicy(policyType, apiPolicy)
	if err != nil {
		resp.Diagnostics.AddError("Error creating AppSec policy", "Could not create policy: "+err.Error())
		return
	}
	policyID := instance.ID

	// Attaching projects is a second call against the policy created above. Terraform records no
	// state when Create reports an error, so anything that fails from here on has to take the new
	// policy with it — otherwise it survives untracked and the next apply creates a duplicate.
	if !plan.ProjectsIds.IsNull() && !plan.ProjectsIds.IsUnknown() {
		if err := r.apiClient.SetShiftLeftPolicyProjects(policyType, policyID, tfconv.SetToStringSlice(plan.ProjectsIds)); err != nil {
			r.rollbackCreatedPolicy(ctx, policyType, policyID, &resp.Diagnostics)
			resp.Diagnostics.AddError("Error setting AppSec policy projects", err.Error())
			return
		}
	}

	// Always re-read: create responses can echo an empty scm_posture scope even when the request
	// carried ids (backend soft-drops unknown/unlinked units). Read-back is also what surfaces a
	// successful project attach.
	instance, err = r.apiClient.GetShiftLeftPolicy(policyType, policyID)
	if err == nil && instance == nil {
		err = fmt.Errorf("policy %s/%s could not be read back after create", policyType, policyID)
	}
	if err != nil {
		r.rollbackCreatedPolicy(ctx, policyType, policyID, &resp.Diagnostics)
		resp.Diagnostics.AddError("Error reading AppSec policy after create", err.Error())
		return
	}
	if err := verifyScmPostureScopeApplied(&plan, instance); err != nil {
		r.rollbackCreatedPolicy(ctx, policyType, policyID, &resp.Diagnostics)
		resp.Diagnostics.AddError("Error applying AppSec policy scope", err.Error())
		return
	}

	state := stateFromPlanAfterWrite(&plan, instance)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *shiftLeftPolicyResource) rollbackCreatedPolicy(ctx context.Context, policyType, policyID string, diags *diag.Diagnostics) {
	if policyID == "" {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Rolling back AppSec policy %s/%s after a failed create", policyType, policyID))
	if err := r.apiClient.DeleteShiftLeftPolicy(policyType, policyID); err != nil {
		diags.AddWarning(
			"Orphaned AppSec policy",
			fmt.Sprintf("Policy %s/%s was created but the apply failed afterwards, and deleting it during "+
				"rollback also failed: %s. It exists in Orca but not in Terraform state — delete it manually "+
				"or import it.", policyType, policyID, err.Error()),
		)
	}
}

// restorePriorPolicy rewrites the remote policy back to prior after a partial Update.
// restoreProjects is true when the projects endpoint already accepted the new set.
func (r *shiftLeftPolicyResource) restorePriorPolicy(ctx context.Context, prior *shiftLeftPolicyResourceModel, restoreProjects bool, diags *diag.Diagnostics) {
	if prior == nil || prior.ID.ValueString() == "" {
		return
	}
	policyType := prior.Type.ValueString()
	policyID := prior.ID.ValueString()
	tflog.Info(ctx, fmt.Sprintf("Restoring AppSec policy %s/%s after a partial update", policyType, policyID))

	apiPolicy, d := planToAPI(prior)
	if d.HasError() {
		diags.AddWarning(
			"Failed to restore AppSec policy after a partial update",
			fmt.Sprintf("Could not rebuild the prior policy body for %s/%s: %s. The remote policy may not match Terraform state — run terraform refresh and reconcile manually.",
				policyType, policyID, d.Errors()),
		)
		return
	}
	apiPolicy.ProjectsIds = nil
	catalogDiags := diag.Diagnostics{}
	if !r.applyCatalog(prior, &apiPolicy, &catalogDiags) {
		diags.AddWarning(
			"Failed to restore AppSec policy after a partial update",
			fmt.Sprintf("Could not re-apply catalog enrichment for %s/%s: %s. The remote policy may not match Terraform state — run terraform refresh and reconcile manually.",
				policyType, policyID, catalogDiags.Errors()),
		)
		return
	}
	if _, err := r.apiClient.UpdateShiftLeftPolicy(policyType, policyID, apiPolicy); err != nil {
		diags.AddWarning(
			"Failed to restore AppSec policy after a partial update",
			fmt.Sprintf("Restoring the prior body of %s/%s failed: %s. The remote policy may not match Terraform state — run terraform refresh and reconcile manually.",
				policyType, policyID, err.Error()),
		)
		return
	}
	if restoreProjects && tfconv.Known(prior.ProjectsIds) {
		if err := r.apiClient.SetShiftLeftPolicyProjects(policyType, policyID, tfconv.SetToStringSlice(prior.ProjectsIds)); err != nil {
			diags.AddWarning(
				"Failed to restore AppSec policy projects after a partial update",
				fmt.Sprintf("Restoring projects on %s/%s failed: %s. The remote policy may not match Terraform state — run terraform refresh and reconcile manually.",
					policyType, policyID, err.Error()),
			)
		}
	}
}

func (r *shiftLeftPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state shiftLeftPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyType := state.Type.ValueString()
	policyID := state.ID.ValueString()

	instance, err := r.apiClient.GetShiftLeftPolicy(policyType, policyID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading AppSec policy", fmt.Sprintf("Could not read policy %s/%s: %s", policyType, policyID, err.Error()))
		return
	}
	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("AppSec policy %s/%s is missing on the remote side.", policyType, policyID))
		resp.State.RemoveResource(ctx)
		return
	}

	newState := apiToState(instance, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *shiftLeftPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan shiftLeftPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state shiftLeftPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Builtin.ValueBool() {
		if field, changed := builtinLockedFieldChanged(&plan, &state); changed {
			resp.Diagnostics.AddError(
				"Cannot modify built-in policy",
				fmt.Sprintf("Field %q is immutable on built-in Orca policies (the API locks it); other fields such as disabled, warn_mode, priority_failure_threshold, control overrides and projects_ids can be changed.", field),
			)
			return
		}
	}

	apiPolicy, diags := planToAPI(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiPolicy.ProjectsIds = nil // detach-all via dedicated endpoint, not main body omitempty.

	policyType := plan.Type.ValueString()
	policyID := plan.ID.ValueString()
	if !r.applyCatalog(&plan, &apiPolicy, &resp.Diagnostics) {
		return
	}

	_, err := r.apiClient.UpdateShiftLeftPolicy(policyType, policyID, apiPolicy)
	if err != nil {
		resp.Diagnostics.AddError("Error updating AppSec policy", "Could not update policy: "+err.Error())
		return
	}

	// Sync projects only when known and actually changed; null/unknown means leave as-is.
	// SetShiftLeftPolicyProjects replaces the whole attachment set, so re-sending an
	// unchanged set is a needless detach/reattach on every apply.
	projectsChanged := tfconv.Known(plan.ProjectsIds) && !plan.ProjectsIds.Equal(state.ProjectsIds)
	if projectsChanged {
		if err := r.apiClient.SetShiftLeftPolicyProjects(policyType, policyID, tfconv.SetToStringSlice(plan.ProjectsIds)); err != nil {
			// Body PUT already landed; restore the prior body so live matches the still-current state.
			r.restorePriorPolicy(ctx, &state, false, &resp.Diagnostics)
			resp.Diagnostics.AddError("Error updating AppSec policy projects", err.Error())
			return
		}
	}

	instance, err := r.apiClient.GetShiftLeftPolicy(policyType, policyID)
	// The write already succeeded, so a missing policy here is a failed read-back, not a deletion.
	if err == nil && instance == nil {
		err = fmt.Errorf("policy %s/%s could not be read back after the update; run terraform refresh", policyType, policyID)
	}
	if err != nil {
		r.restorePriorPolicy(ctx, &state, projectsChanged, &resp.Diagnostics)
		resp.Diagnostics.AddError("Error reading AppSec policy after update", err.Error())
		return
	}
	if err := verifyScmPostureScopeApplied(&plan, instance); err != nil {
		r.restorePriorPolicy(ctx, &state, projectsChanged, &resp.Diagnostics)
		resp.Diagnostics.AddError("Error applying AppSec policy scope", err.Error())
		return
	}

	newState := stateFromPlanAfterWrite(&plan, instance)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *shiftLeftPolicyResource) applyCatalog(plan *shiftLeftPolicyResourceModel, apiPolicy *api_client.ShiftLeftPolicy, diags *diag.Diagnostics) bool {
	catalogType := policyTypeHandlers[plan.Type.ValueString()].catalogType
	if catalogType == "" {
		return true
	}
	if err := r.apiClient.AddAllCatalogControls(catalogType, apiPolicy, allControlsScopeKeys(plan)); err != nil {
		diags.AddError("Error expanding catalog controls", err.Error())
		return false
	}
	if err := r.apiClient.EnrichShiftLeftPolicyFromCatalog(catalogType, apiPolicy); err != nil {
		diags.AddError("Error enriching AppSec policy from catalog", err.Error())
		return false
	}
	return true
}

func (r *shiftLeftPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state shiftLeftPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Builtin.ValueBool() {
		resp.Diagnostics.AddError(
			"Cannot delete built-in policy",
			"Built-in Orca policies cannot be deleted via Terraform.",
		)
		return
	}

	err := r.apiClient.DeleteShiftLeftPolicy(state.Type.ValueString(), state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return
		}
		resp.Diagnostics.AddError("Error deleting AppSec policy", "Could not delete policy: "+err.Error())
	}
}
