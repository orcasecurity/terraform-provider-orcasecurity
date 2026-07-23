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

var (
	_ resource.Resource                = &gitlabGroupResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupResource{}
	_ resource.ResourceWithImportState = &gitlabGroupResource{}
)

var gitlabLabels = shift_left_integration.NewAdoptLabels("GitLab group")

type gitlabGroupResource struct {
	apiClient *api_client.APIClient
}

func NewResource() resource.Resource { return &gitlabGroupResource{} }

func (r *gitlabGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_gitlab_group"
}

func (r *gitlabGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.apiClient = shift_left_integration.ConfigureAPIClient(req)
}

func (r *gitlabGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceSchema()
}

func (r *gitlabGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// <installation_id>/<orca_uuid> or <installation_id>/<gitlab_group_id>
	parts := splitImport(req.ID)
	if parts == nil {
		resp.Diagnostics.AddError("Invalid import ID", "expected <installation_id>/<group_uuid_or_gitlab_group_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("installation_id"), parts[0])...)
	if shift_left_integration.LooksLikeUUID(parts[1]) {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
		return
	}
	n, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "right-hand side must be an Orca group UUID or numeric gitlab_group_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("gitlab_group_id"), n)...)
}

func splitImport(id string) []string {
	for i := 0; i < len(id); i++ {
		if id[i] == '/' {
			left, right := id[:i], id[i+1:]
			if left == "" || right == "" {
				return nil
			}
			return []string{left, right}
		}
	}
	return nil
}

func (r *gitlabGroupResource) ops() shift_left_integration.AdoptedUnitOps[api_client.GitlabGroup, resourceModel] {
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
				return r.apiClient.GetGitlabGroup(iid, id)
			}
			return r.apiClient.FindGitlabGroupByGitlabID(iid, m.GitlabGroupID.ValueInt64())
		},
		Update: func(m *resourceModel, current *api_client.GitlabGroup, body api_client.ScmInstallationUpdate) (*api_client.GitlabGroup, error) {
			return r.apiClient.UpdateGitlabGroup(m.InstallationID.ValueString(), current.ID, body)
		},
		Integrate: func(m *resourceModel, body api_client.ScmInstallationUpdate) error {
			return r.apiClient.IntegrateGitlabUnit(api_client.GitlabUnitIntegrate{
				InstallationID: m.InstallationID.ValueString(),
				GitlabGroupID:  m.GitlabGroupID.ValueInt64(),
				Body:           body,
			})
		},
		Delete: func(m *resourceModel) error {
			id := m.ID.ValueString()
			if id == "" {
				g, err := r.apiClient.FindGitlabGroupByGitlabID(m.InstallationID.ValueString(), m.GitlabGroupID.ValueInt64())
				if err != nil {
					return err
				}
				if g == nil {
					return nil
				}
				id = g.ID
			}
			return r.apiClient.DeleteGitlabGroup(m.InstallationID.ValueString(), id)
		},
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

func (r *gitlabGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.ops().DoCreate(ctx, req, resp)
}

func (r *gitlabGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.ops().DoRead(ctx, req, resp)
}

func (r *gitlabGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.ops().DoUpdate(ctx, req, resp)
}

func (r *gitlabGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.ops().DoDelete(ctx, req, resp)
}
