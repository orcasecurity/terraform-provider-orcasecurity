package shift_left_bitbucket_account

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                = &bitbucketAccountResource{}
	_ resource.ResourceWithConfigure   = &bitbucketAccountResource{}
	_ resource.ResourceWithImportState = &bitbucketAccountResource{}
)

var bitbucketLabels = shift_left_integration.NewAdoptLabels("Bitbucket account")

type bitbucketAccountResource struct {
	apiClient *api_client.APIClient
}

func NewResource() resource.Resource { return &bitbucketAccountResource{} }

func (r *bitbucketAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_bitbucket_account"
}

func (r *bitbucketAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.apiClient = shift_left_integration.ConfigureAPIClient(req)
}

func (r *bitbucketAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceSchema()
}

func (r *bitbucketAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	shift_left_integration.ImportSlashPair(ctx, req, resp, "installation_id", "account_id", "<installation_id>/<account_id>")
}

func (r *bitbucketAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	installationID := plan.InstallationID.ValueString()
	accountID := plan.AccountID.ValueString()
	acc := r.adopt(&resp.Diagnostics, installationID, accountID, plan, config,
		fmt.Sprintf("Account %q on installation %q does not exist. Integrate the Orca Bitbucket account first, then import.", accountID, installationID),
		"Error configuring Bitbucket account")
	if acc == nil {
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
	installationID := state.InstallationID.ValueString()
	accountID := state.AccountID.ValueString()
	acc := shift_left_integration.ReadUnit(ctx, &resp.Diagnostics, bitbucketLabels, accountID,
		func() (*api_client.BitbucketAccount, error) {
			return r.apiClient.GetBitbucketAccount(installationID, accountID)
		},
		resp.State.RemoveResource,
	)
	if acc == nil {
		return
	}
	newState := apiToState(acc)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *bitbucketAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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
		"Error updating Bitbucket account")
	if acc == nil {
		return
	}
	state := apiToState(acc)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *bitbucketAccountResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	shift_left_integration.DeleteNoop(ctx, bitbucketLabels)
}

func (r *bitbucketAccountResource) adopt(
	diags *diag.Diagnostics, installationID, accountID string, plan, config resourceModel, notFoundMsg, writeTitle string,
) *api_client.BitbucketAccount {
	return shift_left_integration.AdoptWrite(
		diags, bitbucketLabels, notFoundMsg, writeTitle,
		func() (*api_client.BitbucketAccount, error) {
			return r.apiClient.GetBitbucketAccount(installationID, accountID)
		},
		func(body api_client.ScmInstallationUpdate) (*api_client.BitbucketAccount, error) {
			return r.apiClient.UpdateBitbucketAccount(installationID, accountID, body)
		},
		func(u *api_client.BitbucketAccount) shift_left_integration.ExistingUnit {
			return shift_left_integration.ExistingFromAPI(u.InstallationMode, u.DefaultPolicies, u.Policies, u.Project, u.ConfigSettings)
		},
		plan.InstallationMode, plan.DefaultPolicies, plan.PoliciesIds, plan.ConfigSettings,
		shift_left_integration.ProjectIntentFrom(config.ProjectID, config.PoliciesIds, config.DefaultPolicies),
	)
}
