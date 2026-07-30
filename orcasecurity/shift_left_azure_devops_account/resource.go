package shift_left_azure_devops_account

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var azureLabels = shift_left_integration.NewAdoptLabels("Azure DevOps account")

func NewResource() resource.Resource {
	return &shift_left_integration.GenericResource[shift_left_integration.AdoptedUnitOps[api_client.AzureDevopsAccount, resourceModel]]{
		TypeNameSuffix: "_shift_left_azure_devops_account",
		SchemaFn:       resourceSchema,
		ImportFn: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			shift_left_integration.ImportScopedUnit(ctx, req, resp, "account_name", "<installation_id>/<account_name_or_orca_uuid>")
		},
		OpsFn: newOps,
	}
}

func newOps(apiClient *api_client.APIClient) shift_left_integration.AdoptedUnitOps[api_client.AzureDevopsAccount, resourceModel] {
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
				return apiClient.GetAzureDevopsAccount(iid, id)
			}
			return apiClient.FindAzureDevopsAccountByName(iid, m.AccountName.ValueString())
		},
		Update: func(m *resourceModel, current *api_client.AzureDevopsAccount, body api_client.ScmInstallationUpdate) (*api_client.AzureDevopsAccount, error) {
			return apiClient.UpdateAzureDevopsAccount(m.InstallationID.ValueString(), current.ID, body)
		},
		Integrate: func(m *resourceModel, body api_client.ScmInstallationUpdate) error {
			return apiClient.IntegrateAzureDevopsUnit(api_client.AzureDevopsUnitIntegrate{
				InstallationID: m.InstallationID.ValueString(),
				AccountName:    m.AccountName.ValueString(),
				Body:           body,
			})
		},
		Delete: func(m *resourceModel) error { return deleteAccount(apiClient, m) },
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

// deleteAccount resolves Orca id from account name when state lacks id (post-import).
func deleteAccount(apiClient *api_client.APIClient, m *resourceModel) error {
	return shift_left_integration.DeleteByLookup(
		m.ID.ValueString(),
		func() (*api_client.AzureDevopsAccount, error) {
			return apiClient.FindAzureDevopsAccountByName(m.InstallationID.ValueString(), m.AccountName.ValueString())
		},
		func(a *api_client.AzureDevopsAccount) string { return a.ID },
		func(id string) error { return apiClient.DeleteAzureDevopsAccount(m.InstallationID.ValueString(), id) },
	)
}
