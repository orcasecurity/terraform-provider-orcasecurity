package shift_left_bitbucket_account

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &bitbucketAccountResource{}
	_ resource.ResourceWithConfigure   = &bitbucketAccountResource{}
	_ resource.ResourceWithImportState = &bitbucketAccountResource{}
)

type bitbucketAccountResource struct {
	apiClient *api_client.APIClient
}

func NewResource() resource.Resource { return &bitbucketAccountResource{} }

func (r *bitbucketAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_bitbucket_account"
}

func (r *bitbucketAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *bitbucketAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceSchema()
}

func (r *bitbucketAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", "expected <installation_id>/<account_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("installation_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_id"), parts[1])...)
}

func (r *bitbucketAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	installationID := plan.InstallationID.ValueString()
	accountID := plan.AccountID.ValueString()
	existing, err := r.apiClient.GetBitbucketAccount(installationID, accountID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Bitbucket account", err.Error())
		return
	}
	if existing == nil {
		resp.Diagnostics.AddError("Bitbucket account not found",
			fmt.Sprintf("Account %q on installation %q does not exist. Integrate the Orca Bitbucket account first, then import.", accountID, installationID))
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
	acc, err := r.apiClient.UpdateBitbucketAccount(installationID, accountID, ad.Body)
	if err != nil {
		resp.Diagnostics.AddError("Error configuring Bitbucket account", err.Error())
		return
	}
	if acc == nil {
		resp.Diagnostics.AddError(
			"Error reading bitbucket account after write",
			"The bitbucket account was configured but could not be read back; the API may not have propagated the change yet. Re-run terraform apply.",
		)
		return
	}
	state := apiToState(acc)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *bitbucketAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	acc, err := r.apiClient.GetBitbucketAccount(state.InstallationID.ValueString(), state.AccountID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Bitbucket account", err.Error())
		return
	}
	if acc == nil {
		tflog.Warn(ctx, fmt.Sprintf("Bitbucket account %s missing remotely", state.AccountID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}
	newState := apiToState(acc)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *bitbucketAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	installationID := plan.InstallationID.ValueString()
	accountID := plan.AccountID.ValueString()
	current, err := r.apiClient.GetBitbucketAccount(installationID, accountID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Bitbucket account before update", err.Error())
		return
	}
	if current == nil {
		resp.Diagnostics.AddError("Bitbucket account not found",
			fmt.Sprintf("Account %q on installation %q was not found. It may have been removed; re-import.", accountID, installationID))
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
	acc, err := r.apiClient.UpdateBitbucketAccount(plan.InstallationID.ValueString(), plan.AccountID.ValueString(), ad.Body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Bitbucket account", err.Error())
		return
	}
	if acc == nil {
		resp.Diagnostics.AddError(
			"Error reading bitbucket account after write",
			"The bitbucket account was configured but could not be read back; the API may not have propagated the change yet. Re-run terraform apply.",
		)
		return
	}
	state := apiToState(acc)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state only. The Bitbucket integrated
// account is not owned by Terraform (created by integrating the Orca
// Bitbucket app) and is left untouched.
func (r *bitbucketAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Removing Bitbucket account from state; the live integration is left untouched.")
}
