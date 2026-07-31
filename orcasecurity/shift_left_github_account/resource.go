package shift_left_github_account

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var githubLabels = shift_left_integration.NewAdoptLabels("GitHub account")

func NewResource() resource.Resource {
	return &shift_left_integration.GenericResource{
		TypeNameSuffix: "_shift_left_github_account",
		SchemaFn:       resourceSchema,
		ImportFn: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			resource.ImportStatePassthroughID(ctx, path.Root("account_id"), req, resp)
		},
		OpsFn: func(apiClient *api_client.APIClient) shift_left_integration.UnitOps {
			return shift_left_integration.AdoptedUnitOps[api_client.GithubInstallation, resourceModel]{
				Labels: githubLabels,
				UnitID: func(m *resourceModel) string { return m.AccountID.ValueString() },
				Get: func(m *resourceModel) (*api_client.GithubInstallation, error) {
					return apiClient.GetGithubInstallation(m.AccountID.ValueString())
				},
				Update: func(m *resourceModel, current *api_client.GithubInstallation, body api_client.ScmInstallationUpdate) (*api_client.GithubInstallation, error) {
					return apiClient.UpdateGithubInstallation(current.ID, body)
				},
				// GitHub units come from the App install callback — there is no
				// Integrate POST, so Create always takes the adopt (PUT) path.
				Integrate: nil,
				Delete: func(m *resourceModel) error {
					return apiClient.DeleteGithubInstallation(m.AccountID.ValueString())
				},
				Snapshot: func(u *api_client.GithubInstallation) shift_left_integration.ExistingUnit {
					return shift_left_integration.ExistingFromCommon(u.ScmUnitCommonFields)
				},
				ToState: apiToState,
				Config:  func(m *resourceModel) *shift_left_integration.ScmConfigFields { return &m.ScmConfigFields },
				Describe: func(m *resourceModel) string {
					return fmt.Sprintf("Account %q", m.AccountID.ValueString())
				},
				CreateHint:       "Install the Orca GitHub App first (UI / GitHub App flow), then import or reference the account_id.",
				CreateErrorTitle: "Error configuring GitHub account",
				UpdateErrorTitle: "Error updating GitHub account",
				DeleteErrorTitle: "Error deleting GitHub account",
			}
		},
	}
}
