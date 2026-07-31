package shift_left_bitbucket_account

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var bitbucketLabels = shift_left_integration.NewAdoptLabels("Bitbucket account")

func NewResource() resource.Resource {
	return &shift_left_integration.GenericResource{
		TypeNameSuffix: "_shift_left_bitbucket_account",
		SchemaFn:       resourceSchema,
		ImportFn: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			shift_left_integration.ImportScopedUnit(ctx, req, resp, "account_id", "<installation_id>/<account_slug_or_orca_uuid>")
		},
		OpsFn: newOps,
	}
}

func newOps(apiClient *api_client.APIClient) shift_left_integration.UnitOps {
	return shift_left_integration.AdoptedUnitOps[api_client.BitbucketAccount, resourceModel]{
		Labels: bitbucketLabels,
		UnitID: func(m *resourceModel) string {
			if m.ID.ValueString() != "" {
				return m.ID.ValueString()
			}
			return m.AccountID.ValueString()
		},
		Get: func(m *resourceModel) (*api_client.BitbucketAccount, error) {
			iid := m.InstallationID.ValueString()
			if id := m.ID.ValueString(); id != "" {
				return apiClient.GetBitbucketAccount(iid, id)
			}
			return apiClient.FindBitbucketAccountBySlug(iid, m.AccountID.ValueString())
		},
		Update: func(m *resourceModel, current *api_client.BitbucketAccount, body api_client.ScmInstallationUpdate) (*api_client.BitbucketAccount, error) {
			return apiClient.UpdateBitbucketAccount(m.InstallationID.ValueString(), current.ID, body)
		},
		Integrate: func(m *resourceModel, body api_client.ScmInstallationUpdate) error {
			return apiClient.IntegrateBitbucketUnit(api_client.BitbucketUnitIntegrate{
				InstallationID: m.InstallationID.ValueString(),
				AccountID:      m.AccountID.ValueString(),
				Body:           body,
			})
		},
		Delete: func(m *resourceModel) error { return deleteAccount(apiClient, m) },
		Snapshot: func(u *api_client.BitbucketAccount) shift_left_integration.ExistingUnit {
			return shift_left_integration.ExistingFromCommon(u.ScmUnitCommonFields)
		},
		ToState: apiToState,
		Config:  func(m *resourceModel) *shift_left_integration.ScmConfigFields { return &m.ScmConfigFields },
		Describe: func(m *resourceModel) string {
			return fmt.Sprintf("Account %q on installation %q", m.AccountID.ValueString(), m.InstallationID.ValueString())
		},
		CreateHint:       "Install the Orca Bitbucket parent connection first (orcasecurity_shift_left_bitbucket_installation).",
		CreateErrorTitle: "Error creating/configuring Bitbucket account",
		UpdateErrorTitle: "Error updating Bitbucket account",
		DeleteErrorTitle: "Error deleting Bitbucket account",
	}
}

// deleteAccount resolves Orca id from account slug when state lacks id (post-import).
func deleteAccount(apiClient *api_client.APIClient, m *resourceModel) error {
	return shift_left_integration.DeleteByLookup(
		m.ID.ValueString(),
		func() (*api_client.BitbucketAccount, error) {
			return apiClient.FindBitbucketAccountBySlug(m.InstallationID.ValueString(), m.AccountID.ValueString())
		},
		func(a *api_client.BitbucketAccount) string { return a.ID },
		func(id string) error { return apiClient.DeleteBitbucketAccount(m.InstallationID.ValueString(), id) },
	)
}
