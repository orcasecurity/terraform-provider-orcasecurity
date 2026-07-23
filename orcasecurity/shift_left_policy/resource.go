package shift_left_policy

import (
	"context"
	"fmt"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &shiftLeftPolicyResource{}
	_ resource.ResourceWithConfigure   = &shiftLeftPolicyResource{}
	_ resource.ResourceWithImportState = &shiftLeftPolicyResource{}
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
	if !r.applyCatalog(&plan, &apiPolicy, &resp.Diagnostics) {
		return
	}

	instance, err := r.apiClient.CreateShiftLeftPolicy(policyType, apiPolicy)
	if err != nil {
		resp.Diagnostics.AddError("Error creating AppSec policy", "Could not create policy: "+err.Error())
		return
	}

	state := stateFromPlanAfterWrite(&plan, instance)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
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

	// Project associations are managed through the dedicated projects endpoint,
	// never the main policy body: projects_ids is omitempty there, so an empty
	// slice is dropped and detach-all (N->0) would be impossible.
	apiPolicy.ProjectsIds = nil

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

	// Sync projects only when the user manages them (known value). An
	// unknown/null projects_ids means "leave associations as-is".
	if !plan.ProjectsIds.IsNull() && !plan.ProjectsIds.IsUnknown() {
		if err := r.apiClient.SetShiftLeftPolicyProjects(policyType, policyID, stringSliceFromSet(plan.ProjectsIds)); err != nil {
			resp.Diagnostics.AddError("Error updating AppSec policy projects", err.Error())
			return
		}
	}

	instance, err := r.apiClient.GetShiftLeftPolicy(policyType, policyID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading AppSec policy after update", err.Error())
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
