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
	return &shift_left_integration.GenericResource[shift_left_integration.AdoptedUnitOps[api_client.GitlabGroup, resourceModel]]{
		TypeNameSuffix: "_shift_left_gitlab_group",
		SchemaFn:       resourceSchema,
		ImportFn:       importState,
		OpsFn:          newOps,
	}
}

// newOps binds the group's CRUD to a client. It is a package-level function rather
// than a closure inside NewResource so that each operation below nests one level
// shallower.
func newOps(apiClient *api_client.APIClient) shift_left_integration.AdoptedUnitOps[api_client.GitlabGroup, resourceModel] {
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
			return apiClient.IntegrateGitlabUnit(api_client.GitlabUnitIntegrate{
				InstallationID: m.InstallationID.ValueString(),
				GitlabGroupID:  m.GitlabGroupID.ValueInt64(),
				Body:           body,
			})
		},
		Delete: func(m *resourceModel) error { return deleteGroup(apiClient, m) },
		Snapshot: func(u *api_client.GitlabGroup) shift_left_integration.ExistingUnit {
			return shift_left_integration.ExistingFromCommon(u.ScmUnitCommonFields)
		},
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

// deleteGroup resolves the Orca id before deleting: state may carry only the
// numeric GitLab group id (the first destroy after an import), and a group that no
// longer resolves is already gone, which is a successful delete rather than an error.
func deleteGroup(apiClient *api_client.APIClient, m *resourceModel) error {
	id := m.ID.ValueString()
	if id == "" {
		g, err := apiClient.FindGitlabGroupByGitlabID(m.InstallationID.ValueString(), m.GitlabGroupID.ValueInt64())
		if err != nil {
			return err
		}
		if g == nil {
			return nil
		}
		id = g.ID
	}
	return apiClient.DeleteGitlabGroup(m.InstallationID.ValueString(), id)
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
