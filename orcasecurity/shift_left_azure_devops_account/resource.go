package shift_left_azure_devops_account

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                = &azureDevopsAccountResource{}
	_ resource.ResourceWithConfigure   = &azureDevopsAccountResource{}
	_ resource.ResourceWithImportState = &azureDevopsAccountResource{}
)

var azureLabels = shift_left_integration.NewAdoptLabels("Azure DevOps account")

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

func (r *azureDevopsAccountResource) ops() shift_left_integration.AdoptedUnitOps[api_client.AzureDevopsAccount, resourceModel] {
	return shift_left_integration.AdoptedUnitOps[api_client.AzureDevopsAccount, resourceModel]{
		Labels: azureLabels,
		UnitID: func(m *resourceModel) string { return m.AccountID.ValueString() },
		Get: func(m *resourceModel) (*api_client.AzureDevopsAccount, error) {
			return r.apiClient.GetAzureDevopsAccount(m.InstallationID.ValueString(), m.AccountID.ValueString())
		},
		Update: func(m *resourceModel, body api_client.ScmInstallationUpdate) (*api_client.AzureDevopsAccount, error) {
			return r.apiClient.UpdateAzureDevopsAccount(m.InstallationID.ValueString(), m.AccountID.ValueString(), body)
		},
		Snapshot: func(u *api_client.AzureDevopsAccount) shift_left_integration.ExistingUnit {
			return shift_left_integration.ExistingFromCommon(u.ScmUnitCommonFields)
		},
		ToState: apiToState,
		Config:  func(m *resourceModel) *shift_left_integration.ScmConfigFields { return &m.ScmConfigFields },
		Describe: func(m *resourceModel) string {
			return fmt.Sprintf("Account %q on installation %q", m.AccountID.ValueString(), m.InstallationID.ValueString())
		},
		CreateHint:       "Integrate the Orca Azure DevOps account first, then import.",
		CreateErrorTitle: "Error configuring Azure DevOps account",
		UpdateErrorTitle: "Error updating Azure DevOps account",
	}
}

func (r *azureDevopsAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.ops().DoCreate(ctx, req, resp)
}

func (r *azureDevopsAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.ops().DoRead(ctx, req, resp)
}

func (r *azureDevopsAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.ops().DoUpdate(ctx, req, resp)
}

func (r *azureDevopsAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.ops().DoDelete(ctx, req, resp)
}
