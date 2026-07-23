package shift_left_azure_devops_account

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                = &azureDevopsAccountResource{}
	_ resource.ResourceWithConfigure   = &azureDevopsAccountResource{}
	_ resource.ResourceWithImportState = &azureDevopsAccountResource{}
)

var azureLabels = shift_left_integration.AdoptLabels{
	NotFoundTitle:  "Azure DevOps account not found",
	NilReadTitle:   "Error reading azure devops account after write",
	NilReadDetail:  "The azure devops account was configured but could not be read back; the API may not have propagated the change yet. Re-run terraform apply.",
	ReadErrorTitle: "Error reading Azure DevOps account",
	DeleteLog:      "Removing Azure DevOps account from state; the live integration is left untouched.",
	MissingWarn:    "Azure DevOps account %s missing remotely",
}

type azureDevopsAccountResource struct {
	apiClient *api_client.APIClient
}

func NewResource() resource.Resource { return &azureDevopsAccountResource{} }

func (r *azureDevopsAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_azure_devops_account"
}

func (r *azureDevopsAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.apiClient = shift_left_integration.ConfigureAPIClient(req)
}

func (r *azureDevopsAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceSchema()
}

func (r *azureDevopsAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	shift_left_integration.ImportSlashPair(ctx, req, resp, "installation_id", "account_id", "<installation_id>/<account_id>")
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
	acc := r.adopt(&resp.Diagnostics, installationID, accountID, plan, config,
		fmt.Sprintf("Account %q on installation %q does not exist. Integrate the Orca Azure DevOps account first, then import.", accountID, installationID),
		"Error configuring Azure DevOps account")
	if acc == nil {
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
	installationID := state.InstallationID.ValueString()
	accountID := state.AccountID.ValueString()
	acc := shift_left_integration.ReadUnit(ctx, &resp.Diagnostics, azureLabels, accountID,
		func() (*api_client.AzureDevopsAccount, error) {
			return r.apiClient.GetAzureDevopsAccount(installationID, accountID)
		},
		resp.State.RemoveResource,
	)
	if acc == nil {
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
	acc := r.adopt(&resp.Diagnostics, installationID, accountID, plan, config,
		fmt.Sprintf("Account %q on installation %q was not found. It may have been removed; re-import.", accountID, installationID),
		"Error updating Azure DevOps account")
	if acc == nil {
		return
	}
	state := apiToState(acc)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *azureDevopsAccountResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	shift_left_integration.DeleteNoop(ctx, azureLabels)
}

func (r *azureDevopsAccountResource) adopt(
	diags *diag.Diagnostics, installationID, accountID string, plan, config resourceModel, notFoundMsg, writeTitle string,
) *api_client.AzureDevopsAccount {
	return shift_left_integration.AdoptWrite(
		diags, azureLabels, notFoundMsg, writeTitle,
		func() (*api_client.AzureDevopsAccount, error) {
			return r.apiClient.GetAzureDevopsAccount(installationID, accountID)
		},
		func(body api_client.ScmInstallationUpdate) (*api_client.AzureDevopsAccount, error) {
			return r.apiClient.UpdateAzureDevopsAccount(installationID, accountID, body)
		},
		func(u *api_client.AzureDevopsAccount) shift_left_integration.ExistingUnit {
			return shift_left_integration.ExistingFromAPI(u.InstallationMode, u.DefaultPolicies, u.Policies, u.Project, u.ConfigSettings)
		},
		plan.InstallationMode, plan.DefaultPolicies, plan.PoliciesIds, plan.ConfigSettings,
		shift_left_integration.ProjectIntentFrom(config.ProjectID, config.PoliciesIds, config.DefaultPolicies),
	)
}
