package shift_left_azure_devops_account

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &azureDevopsAccountResource{}
	_ resource.ResourceWithConfigure   = &azureDevopsAccountResource{}
	_ resource.ResourceWithImportState = &azureDevopsAccountResource{}
)

type azureDevopsAccountResource struct {
	apiClient *api_client.APIClient
}

func NewResource() resource.Resource { return &azureDevopsAccountResource{} }

func (r *azureDevopsAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_azure_devops_account"
}

func (r *azureDevopsAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *azureDevopsAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceSchema()
}

func (r *azureDevopsAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", "expected <installation_id>/<account_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("installation_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_id"), parts[1])...)
}

func (r *azureDevopsAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	installationID := plan.InstallationID.ValueString()
	accountID := plan.AccountID.ValueString()
	acc, err := r.write(installationID, accountID, plan, config)
	if errors.Is(err, shift_left_integration.ErrUnitNotFound) {
		resp.Diagnostics.AddError("Azure DevOps account not found",
			fmt.Sprintf("Account %q on installation %q does not exist. Integrate the Orca Azure DevOps account first, then import.", accountID, installationID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error configuring Azure DevOps account", err.Error())
		return
	}
	if acc == nil {
		resp.Diagnostics.AddError(
			"Error reading azure devops account after write",
			"The azure devops account was configured but could not be read back; the API may not have propagated the change yet. Re-run terraform apply.",
		)
		return
	}
	state := apiToState(acc)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *azureDevopsAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	acc, err := r.apiClient.GetAzureDevopsAccount(state.InstallationID.ValueString(), state.AccountID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Azure DevOps account", err.Error())
		return
	}
	if acc == nil {
		tflog.Warn(ctx, fmt.Sprintf("Azure DevOps account %s missing remotely", state.AccountID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}
	newState := apiToState(acc)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *azureDevopsAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	installationID := plan.InstallationID.ValueString()
	accountID := plan.AccountID.ValueString()
	acc, err := r.write(installationID, accountID, plan, config)
	if errors.Is(err, shift_left_integration.ErrUnitNotFound) {
		resp.Diagnostics.AddError("Azure DevOps account not found",
			fmt.Sprintf("Account %q on installation %q was not found. It may have been removed; re-import.", accountID, installationID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error updating Azure DevOps account", err.Error())
		return
	}
	if acc == nil {
		resp.Diagnostics.AddError(
			"Error reading azure devops account after write",
			"The azure devops account was configured but could not be read back; the API may not have propagated the change yet. Re-run terraform apply.",
		)
		return
	}
	state := apiToState(acc)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state only. The Azure DevOps integrated
// account is not owned by Terraform and is left untouched.
func (r *azureDevopsAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Removing Azure DevOps account from state; the live integration is left untouched.")
}

func (r *azureDevopsAccountResource) write(installationID, accountID string, plan, config resourceModel) (*api_client.AzureDevopsAccount, error) {
	project := shift_left_integration.ProjectIntentFrom(config.ProjectID, config.PoliciesIds, config.DefaultPolicies)
	return shift_left_integration.WriteAdopted(
		func() (*api_client.AzureDevopsAccount, error) {
			return r.apiClient.GetAzureDevopsAccount(installationID, accountID)
		},
		func(body api_client.ScmInstallationUpdate) (*api_client.AzureDevopsAccount, error) {
			return r.apiClient.UpdateAzureDevopsAccount(installationID, accountID, body)
		},
		func(u *api_client.AzureDevopsAccount) shift_left_integration.ExistingUnit {
			return shift_left_integration.ExistingFromAPI(u.InstallationMode, u.DefaultPolicies, u.Policies, u.Project, u.ConfigSettings)
		},
		plan.InstallationMode, plan.DefaultPolicies, plan.PoliciesIds, plan.ConfigSettings, project,
	)
}
