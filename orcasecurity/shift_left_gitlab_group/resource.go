package shift_left_gitlab_group

import (
	"context"
	"fmt"
	"strconv"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var gitlabLabels = shift_left_integration.NewAdoptLabels("GitLab group")

func NewResource() resource.Resource {
	return &shift_left_integration.GenericResource{
		TypeNameSuffix: "_shift_left_gitlab_group",
		SchemaFn:       resourceSchema,
		ImportFn:       importState,
		OpsFn:          newOps,
	}
}

func newOps(apiClient *api_client.APIClient) shift_left_integration.UnitOps {
	return shift_left_integration.AdoptedUnitOps[api_client.GitlabGroup, resourceModel]{
		Labels: gitlabLabels,
		UnitID: func(m *resourceModel) string {
			if m.ID.ValueString() != "" {
				return m.ID.ValueString()
			}
			return fmt.Sprintf("gitlab_group_id=%d", m.GitlabGroupID.ValueInt64())
		},
		Get: func(m *resourceModel) (*api_client.GitlabGroup, error) {
			iid := m.InstallationID.ValueString()
			if id := m.ID.ValueString(); id != "" {
				return apiClient.GetGitlabGroup(iid, id)
			}
			return apiClient.FindGitlabGroupByGitlabID(iid, m.GitlabGroupID.ValueInt64())
		},
		Update: func(m *resourceModel, current *api_client.GitlabGroup, body api_client.ScmInstallationUpdate) (*api_client.GitlabGroup, error) {
			return apiClient.UpdateGitlabGroup(m.InstallationID.ValueString(), current.ID, body)
		},
		Integrate: func(m *resourceModel, body api_client.ScmInstallationUpdate) error {
			// The GitLab integrate endpoint accepts only ALWAYS/NEVER for
			// skip_check_runs; the full enum (ONLY_ON_INTERNAL_ISSUE) is accepted by
			// the update endpoint once the group is integrated. Failing here replaces
			// the backend's raw validation error with an actionable one.
			if body.ConfigSettings.SkipCheckRuns == "ONLY_ON_INTERNAL_ISSUE" {
				return fmt.Errorf(
					"skip_check_runs = %q is not accepted when integrating a new GitLab group "+
						"(the integrate API allows only ALWAYS or NEVER). Integrate with ALWAYS or NEVER "+
						"and switch to ONLY_ON_INTERNAL_ISSUE on a later apply, or import an "+
						"already-integrated group instead", "ONLY_ON_INTERNAL_ISSUE")
			}
			return apiClient.IntegrateGitlabUnit(api_client.GitlabUnitIntegrate{
				InstallationID: m.InstallationID.ValueString(),
				GitlabGroupID:  m.GitlabGroupID.ValueInt64(),
				Body:           body,
			})
		},
		Delete:  func(m *resourceModel) error { return deleteGroup(apiClient, m) },
		ToState: apiToState,
		Config:  func(m *resourceModel) *shift_left_integration.ScmConfigFields { return &m.ScmConfigFields },
		Describe: func(m *resourceModel) string {
			return fmt.Sprintf("GitLab group %d on installation %q", m.GitlabGroupID.ValueInt64(), m.InstallationID.ValueString())
		},
		CreateHint:       "Install the Orca GitLab parent connection first (orcasecurity_shift_left_gitlab_installation).",
		CreateErrorTitle: "Error creating/configuring GitLab group",
		UpdateErrorTitle: "Error updating GitLab group",
		DeleteErrorTitle: "Error deleting GitLab group",
	}
}

// deleteGroup resolves Orca id from SCM group id when state lacks id (post-import).
func deleteGroup(apiClient *api_client.APIClient, m *resourceModel) error {
	return shift_left_integration.DeleteByLookup(
		m.ID.ValueString(),
		func() (*api_client.GitlabGroup, error) {
			return apiClient.FindGitlabGroupByGitlabID(m.InstallationID.ValueString(), m.GitlabGroupID.ValueInt64())
		},
		func(g *api_client.GitlabGroup) string { return g.ID },
		func(id string) error { return apiClient.DeleteGitlabGroup(m.InstallationID.ValueString(), id) },
	)
}

func importState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	rest, resolved := shift_left_integration.ImportScopedInstallation(ctx, req, resp, "<installation_id>/<group_uuid_or_gitlab_group_id>")
	if resolved {
		return
	}
	n, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "right-hand side must be an Orca group UUID or numeric gitlab_group_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("gitlab_group_id"), n)...)
}
