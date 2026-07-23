package shift_left_gitlab_group

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

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
	shift_left_integration.ImportSlashPair(ctx, req, resp, "installation_id", "group_id", "<installation_id>/<group_id>")
}

func (r *gitlabGroupResource) ops() shift_left_integration.AdoptedUnitOps[api_client.GitlabGroup, resourceModel] {
	return shift_left_integration.AdoptedUnitOps[api_client.GitlabGroup, resourceModel]{
		Labels: gitlabLabels,
		UnitID: func(m *resourceModel) string { return m.GroupID.ValueString() },
		Get: func(m *resourceModel) (*api_client.GitlabGroup, error) {
			return r.apiClient.GetGitlabGroup(m.InstallationID.ValueString(), m.GroupID.ValueString())
		},
		Update: func(m *resourceModel, body api_client.ScmInstallationUpdate) (*api_client.GitlabGroup, error) {
			return r.apiClient.UpdateGitlabGroup(m.InstallationID.ValueString(), m.GroupID.ValueString(), body)
		},
		Snapshot: func(u *api_client.GitlabGroup) shift_left_integration.ExistingUnit {
			return shift_left_integration.ExistingFromCommon(u.ScmUnitCommonFields)
		},
		ToState: apiToState,
		Config:  func(m *resourceModel) *shift_left_integration.ScmConfigFields { return &m.ScmConfigFields },
		Describe: func(m *resourceModel) string {
			return fmt.Sprintf("Group %q on installation %q", m.GroupID.ValueString(), m.InstallationID.ValueString())
		},
		CreateHint:       "Integrate the Orca GitLab group first, then import.",
		CreateErrorTitle: "Error configuring GitLab group",
		UpdateErrorTitle: "Error updating GitLab group",
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
