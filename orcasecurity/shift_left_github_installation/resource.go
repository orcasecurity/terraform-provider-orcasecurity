package shift_left_github_installation

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var githubLabels = shift_left_integration.NewAdoptLabels("GitHub installation")

func NewResource() resource.Resource {
	return &shift_left_integration.AdoptedUnitResource[api_client.GithubInstallation, resourceModel]{
		TypeNameSuffix: "_shift_left_github_installation",
		SchemaFn:       resourceSchema,
		ImportFn: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			resource.ImportStatePassthroughID(ctx, path.Root("installation_id"), req, resp)
		},
		OpsFn: func(apiClient *api_client.APIClient) shift_left_integration.AdoptedUnitOps[api_client.GithubInstallation, resourceModel] {
			return shift_left_integration.AdoptedUnitOps[api_client.GithubInstallation, resourceModel]{
				Labels: githubLabels,
				UnitID: func(m *resourceModel) string { return m.InstallationID.ValueString() },
				Get: func(m *resourceModel) (*api_client.GithubInstallation, error) {
					return apiClient.GetGithubInstallation(m.InstallationID.ValueString())
				},
				Update: func(m *resourceModel, current *api_client.GithubInstallation, body api_client.ScmInstallationUpdate) (*api_client.GithubInstallation, error) {
					return apiClient.UpdateGithubInstallation(current.ID, body)
				},
				// nil: GitHub units come from the App install callback.
				Delete: func(m *resourceModel) error {
					return apiClient.DeleteGithubInstallation(m.InstallationID.ValueString())
				},
				Snapshot: func(u *api_client.GithubInstallation) shift_left_integration.ExistingUnit {
					return shift_left_integration.ExistingFromCommon(u.ScmUnitCommonFields)
				},
				ToState: apiToState,
				Config:  func(m *resourceModel) *shift_left_integration.ScmConfigFields { return &m.ScmConfigFields },
				Describe: func(m *resourceModel) string {
					return fmt.Sprintf("Installation %q", m.InstallationID.ValueString())
				},
				CreateHint:       "Install the Orca GitHub App first (UI / GitHub App flow), then import or reference the installation_id.",
				CreateErrorTitle: "Error configuring GitHub installation",
				UpdateErrorTitle: "Error updating GitHub installation",
				DeleteErrorTitle: "Error deleting GitHub installation",
			}
		},
	}
}
