package shift_left_bitbucket_account

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var bitbucketLabels = shift_left_integration.NewAdoptLabels("Bitbucket account")

// integrateGuard: the Bitbucket integrate endpoint requires an explicit
// repositories list under SELECTED_REPOSITORIES and rejects the request
// without one — and this resource intentionally does not model per-repository
// selection (that is orcasecurity_shift_left_bitbucket_repository). Failing
// here replaces the backend's raw validation error with an actionable one.
// Only integrate is affected: updating an existing account to
// SELECTED_REPOSITORIES is accepted.
func integrateGuard(body api_client.ScmInstallationUpdate) error {
	if body.InstallationMode != "SELECTED_REPOSITORIES" {
		return nil
	}
	return fmt.Errorf(
		"integrating a new Bitbucket workspace requires installation_mode = %q: "+
			"the API only accepts SELECTED_REPOSITORIES on integrate together with an explicit "+
			"repository list, which this resource does not send. Integrate with SCAN_ALL_INCLUDE_FUTURE "+
			"(you can switch to SELECTED_REPOSITORIES on a later apply), or import an already-integrated "+
			"account instead", "SCAN_ALL_INCLUDE_FUTURE")
}

func NewResource() resource.Resource {
	return &shift_left_integration.GenericResource{
		TypeNameSuffix: "_shift_left_bitbucket_account",
		SchemaFn:       resourceSchema,
		ImportFn: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			shift_left_common.ImportScopedUnit(ctx, req, resp, "account_id", "<installation_id>/<account_slug_or_orca_uuid>")
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
		IntegrateGuard: integrateGuard,
		Integrate: func(m *resourceModel, body api_client.ScmInstallationUpdate) error {
			return apiClient.IntegrateBitbucketUnit(api_client.BitbucketUnitIntegrate{
				InstallationID: m.InstallationID.ValueString(),
				AccountID:      m.AccountID.ValueString(),
				Body:           body,
			})
		},
		Delete:  func(m *resourceModel) error { return deleteAccount(apiClient, m) },
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
