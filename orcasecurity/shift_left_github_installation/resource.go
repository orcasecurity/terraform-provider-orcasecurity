package shift_left_github_installation

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &githubInstallationResource{}
	_ resource.ResourceWithConfigure   = &githubInstallationResource{}
	_ resource.ResourceWithImportState = &githubInstallationResource{}
)

type githubInstallationResource struct {
	apiClient *api_client.APIClient
}

func NewResource() resource.Resource { return &githubInstallationResource{} }

func (r *githubInstallationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_github_installation"
}

func (r *githubInstallationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *githubInstallationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceSchema()
}

func (r *githubInstallationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("installation_id"), req, resp)
}

func (r *githubInstallationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := plan.InstallationID.ValueString()
	existing, err := r.apiClient.GetGithubInstallation(id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitHub installation", err.Error())
		return
	}
	if existing == nil {
		resp.Diagnostics.AddError("GitHub installation not found",
			fmt.Sprintf("Installation %q does not exist. Install the Orca GitHub App first, then import.", id))
		return
	}
	var config resourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project := shift_left_integration.ProjectIntentFrom(config.ProjectID, config.PoliciesIds, config.DefaultPolicies)
	ad := shift_left_integration.Adopt(plan.InstallationMode, plan.DefaultPolicies, plan.PoliciesIds, plan.ConfigSettings, project, shift_left_integration.ExistingUnit{
		InstallationMode: existing.InstallationMode,
		DefaultPolicies:  existing.DefaultPolicies,
		PolicyIDs:        api_client.PolicyRefIDs(existing.Policies),
		ConfigSettings:   existing.ConfigSettings,
		ProjectID:        api_client.ProjectRefID(existing.Project),
	})
	inst, err := r.apiClient.UpdateGithubInstallation(id, ad.Body)
	if err != nil {
		resp.Diagnostics.AddError("Error configuring GitHub installation", err.Error())
		return
	}
	if inst == nil {
		resp.Diagnostics.AddError(
			"Error reading github installation after write",
			"The github installation was configured but could not be read back; the API may not have propagated the change yet. Re-run terraform apply.",
		)
		return
	}
	state := apiToState(inst)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *githubInstallationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	inst, err := r.apiClient.GetGithubInstallation(state.InstallationID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitHub installation", err.Error())
		return
	}
	if inst == nil {
		tflog.Warn(ctx, fmt.Sprintf("GitHub installation %s missing remotely", state.InstallationID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}
	newState := apiToState(inst)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *githubInstallationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := plan.InstallationID.ValueString()
	current, err := r.apiClient.GetGithubInstallation(id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitHub installation before update", err.Error())
		return
	}
	if current == nil {
		resp.Diagnostics.AddError("GitHub installation not found",
			fmt.Sprintf("Installation %q was not found. It may have been removed; re-import.", id))
		return
	}
	var config resourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project := shift_left_integration.ProjectIntentFrom(config.ProjectID, config.PoliciesIds, config.DefaultPolicies)
	ad := shift_left_integration.Adopt(plan.InstallationMode, plan.DefaultPolicies, plan.PoliciesIds, plan.ConfigSettings, project, shift_left_integration.ExistingUnit{
		InstallationMode: current.InstallationMode,
		DefaultPolicies:  current.DefaultPolicies,
		PolicyIDs:        api_client.PolicyRefIDs(current.Policies),
		ConfigSettings:   current.ConfigSettings,
		ProjectID:        api_client.ProjectRefID(current.Project),
	})
	inst, err := r.apiClient.UpdateGithubInstallation(plan.InstallationID.ValueString(), ad.Body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating GitHub installation", err.Error())
		return
	}
	if inst == nil {
		resp.Diagnostics.AddError(
			"Error reading github installation after write",
			"The github installation was configured but could not be read back; the API may not have propagated the change yet. Re-run terraform apply.",
		)
		return
	}
	state := apiToState(inst)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state only. The GitHub installation is not
// owned by Terraform (created by installing the Orca GitHub App) and is left
// untouched.
func (r *githubInstallationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Removing GitHub installation from state; the live integration is left untouched.")
}
