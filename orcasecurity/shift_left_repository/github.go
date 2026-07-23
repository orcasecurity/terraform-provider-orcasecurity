package shift_left_repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &githubRepositoryResource{}
	_ resource.ResourceWithConfigure   = &githubRepositoryResource{}
	_ resource.ResourceWithImportState = &githubRepositoryResource{}
)

type githubRepositoryResource struct {
	apiClient *api_client.APIClient
}

func NewGithubRepositoryResource() resource.Resource { return &githubRepositoryResource{} }

type githubRepositoryModel struct {
	InstallationID     types.String `tfsdk:"installation_id"`
	GithubRepositoryID types.Int64  `tfsdk:"github_repository_id"`
	RepoConfigFields
}

func (r *githubRepositoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_github_repository"
}

func (r *githubRepositoryResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.apiClient = shift_left_integration.ConfigureAPIClient(req)
}

func (r *githubRepositoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := sharedRepoAttributes("GitHub", fullSkipCheckRuns)
	attrs["installation_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca id of the GitHub installation (see `orcasecurity_shift_left_github_installations`).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["github_repository_id"] = rschema.Int64Attribute{
		Required:      true,
		Description:   "Numeric GitHub repository id.",
		PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
	}
	resp.Schema = rschema.Schema{
		Description: "Integrates a single GitHub repository into Orca Shift Left under an existing GitHub installation. " +
			"Destroying the resource un-integrates the repository (deletes its repository context); it does not touch the repository on GitHub. " +
			"Import with `installation_id:github_repository_id`.",
		Attributes: attrs,
	}
}

func (r *githubRepositoryResource) ops(plan *githubRepositoryModel) repoOps {
	installationID := plan.InstallationID.ValueString()
	githubRepositoryID := plan.GithubRepositoryID.ValueInt64()
	return repoOps{
		client:  r.apiClient,
		scmName: "GitHub",
		integrate: func() error {
			return r.apiClient.IntegrateGithubRepository(api_client.GithubRepositoryIntegrate{
				InstallationID:     installationID,
				GithubRepositoryID: githubRepositoryID,
				Name:               plan.Name.ValueString(),
				URL:                plan.URL.ValueString(),
				Branch:             plan.Branch.ValueString(),
				ProjectID:          plan.ProjectID.ValueString(),
			})
		},
		find: func() (*api_client.ScmRepository, error) {
			return r.apiClient.FindGithubRepository(installationID, githubRepositoryID)
		},
		update: r.apiClient.UpdateGithubRepositories,
	}
}

func githubFields(m *githubRepositoryModel) *RepoConfigFields { return &m.RepoConfigFields }

func (r *githubRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	repoCreate(ctx, req, resp, r.ops, githubFields)
}

func (r *githubRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	repoRead(ctx, req, resp, r.ops, githubFields)
}

func (r *githubRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	repoUpdate(ctx, req, resp, r.ops, githubFields)
}

func (r *githubRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	repoDelete(ctx, req, resp, r.ops, githubFields)
}

func (r *githubRepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	installationID, repoID, ok := strings.Cut(req.ID, ":")
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", "expected format installation_id:github_repository_id")
		return
	}
	numericID, err := strconv.ParseInt(repoID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("github_repository_id must be numeric: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("installation_id"), installationID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("github_repository_id"), numericID)...)
}
