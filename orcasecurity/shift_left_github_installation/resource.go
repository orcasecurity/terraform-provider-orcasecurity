package shift_left_github_installation

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                = &githubInstallationResource{}
	_ resource.ResourceWithConfigure   = &githubInstallationResource{}
	_ resource.ResourceWithImportState = &githubInstallationResource{}
)

var githubLabels = shift_left_integration.NewAdoptLabels("GitHub installation")

type githubInstallationResource struct {
	apiClient *api_client.APIClient
}

func NewResource() resource.Resource { return &githubInstallationResource{} }

func (r *githubInstallationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_github_installation"
}

func (r *githubInstallationResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.apiClient = shift_left_integration.ConfigureAPIClient(req)
}

func (r *githubInstallationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceSchema()
}

func (r *githubInstallationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("installation_id"), req, resp)
}

func (r *githubInstallationResource) ops() shift_left_integration.AdoptedUnitOps[api_client.GithubInstallation, resourceModel] {
	return shift_left_integration.AdoptedUnitOps[api_client.GithubInstallation, resourceModel]{
		Labels: githubLabels,
		UnitID: func(m *resourceModel) string { return m.InstallationID.ValueString() },
		Get: func(m *resourceModel) (*api_client.GithubInstallation, error) {
			return r.apiClient.GetGithubInstallation(m.InstallationID.ValueString())
		},
		Update: func(m *resourceModel, current *api_client.GithubInstallation, body api_client.ScmInstallationUpdate) (*api_client.GithubInstallation, error) {
			return r.apiClient.UpdateGithubInstallation(current.ID, body)
		},
		// Integrate is nil: GitHub units are created by the GitHub App install callback.
		Delete: func(m *resourceModel) error {
			return r.apiClient.DeleteGithubInstallation(m.InstallationID.ValueString())
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
}

func (r *githubInstallationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.ops().DoCreate(ctx, req, resp)
}

func (r *githubInstallationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.ops().DoRead(ctx, req, resp)
}

func (r *githubInstallationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.ops().DoUpdate(ctx, req, resp)
}

func (r *githubInstallationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.ops().DoDelete(ctx, req, resp)
}
