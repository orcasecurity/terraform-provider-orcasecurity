package shift_left_azure_devops_account

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/path"
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
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", "expected <installation_id>/<account_name_or_orca_uuid>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("installation_id"), parts[0])...)
	if shift_left_integration.LooksLikeUUID(parts[1]) {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_name"), parts[1])...)
}

func (r *azureDevopsAccountResource) ops() shift_left_integration.AdoptedUnitOps[api_client.AzureDevopsAccount, resourceModel] {
	return shift_left_integration.AdoptedUnitOps[api_client.AzureDevopsAccount, resourceModel]{
		Labels: azureLabels,
		UnitID: func(m *resourceModel) string {
			if m.ID.ValueString() != "" {
				return m.ID.ValueString()
			}
			return m.AccountName.ValueString()
		},
		Get: func(m *resourceModel) (*api_client.AzureDevopsAccount, error) {
			iid := m.InstallationID.ValueString()
			if id := m.ID.ValueString(); id != "" {
				return r.apiClient.GetAzureDevopsAccount(iid, id)
			}
			return r.apiClient.FindAzureDevopsAccountByName(iid, m.AccountName.ValueString())
		},
		Update: func(m *resourceModel, current *api_client.AzureDevopsAccount, body api_client.ScmInstallationUpdate) (*api_client.AzureDevopsAccount, error) {
			return r.apiClient.UpdateAzureDevopsAccount(m.InstallationID.ValueString(), current.ID, body)
		},
		Integrate: func(m *resourceModel, body api_client.ScmInstallationUpdate) error {
			return r.apiClient.IntegrateAzureDevopsUnit(api_client.AzureDevopsUnitIntegrate{
				InstallationID: m.InstallationID.ValueString(),
				AccountName:    m.AccountName.ValueString(),
				Body:           body,
			})
		},
		Delete: func(m *resourceModel) error {
			id := m.ID.ValueString()
			if id == "" {
				a, err := r.apiClient.FindAzureDevopsAccountByName(m.InstallationID.ValueString(), m.AccountName.ValueString())
				if err != nil {
					return err
				}
				if a == nil {
					return nil
				}
				id = a.ID
			}
			return r.apiClient.DeleteAzureDevopsAccount(m.InstallationID.ValueString(), id)
		},
		Snapshot: func(u *api_client.AzureDevopsAccount) shift_left_integration.ExistingUnit {
			return shift_left_integration.ExistingFromCommon(u.ScmUnitCommonFields)
		},
		ToState: apiToState,
		Config:  func(m *resourceModel) *shift_left_integration.ScmConfigFields { return &m.ScmConfigFields },
		Describe: func(m *resourceModel) string {
			return fmt.Sprintf("Account %q on installation %q", m.AccountName.ValueString(), m.InstallationID.ValueString())
		},
		CreateHint:       "Install the Orca Azure DevOps parent connection first (orcasecurity_shift_left_azure_devops_installation).",
		CreateErrorTitle: "Error creating/configuring Azure DevOps account",
		UpdateErrorTitle: "Error updating Azure DevOps account",
		DeleteErrorTitle: "Error deleting Azure DevOps account",
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
