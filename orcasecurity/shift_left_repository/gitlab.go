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
	_ resource.Resource                = &gitlabRepositoryResource{}
	_ resource.ResourceWithConfigure   = &gitlabRepositoryResource{}
	_ resource.ResourceWithImportState = &gitlabRepositoryResource{}
)

type gitlabRepositoryResource struct {
	apiClient *api_client.APIClient
}

func NewGitlabRepositoryResource() resource.Resource { return &gitlabRepositoryResource{} }

type gitlabRepositoryModel struct {
	InstallationID  types.String `tfsdk:"installation_id"`
	GitlabGroupID   types.Int64  `tfsdk:"gitlab_group_id"`
	GitlabProjectID types.Int64  `tfsdk:"gitlab_project_id"`
	RepoConfigFields
}

func (r *gitlabRepositoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_gitlab_repository"
}

func (r *gitlabRepositoryResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.apiClient = shift_left_integration.ConfigureAPIClient(req)
}

func (r *gitlabRepositoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := sharedRepoAttributes("GitLab", gitlabSkipCheckRuns)
	attrs["installation_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca id of the GitLab installation (see `orcasecurity_shift_left_gitlab_installation`).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["gitlab_group_id"] = rschema.Int64Attribute{
		Required: true,
		Description: "Numeric GitLab group id owning the project. If the group is not yet integrated with Orca, " +
			"integrating the first repository also creates the group unit.",
		PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
	}
	attrs["gitlab_project_id"] = rschema.Int64Attribute{
		Required:      true,
		Description:   "Numeric GitLab project (repository) id.",
		PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
	}
	resp.Schema = rschema.Schema{
		Description: "Integrates a single GitLab project (repository) into Orca Shift Left under an existing GitLab installation. " +
			"Destroying the resource un-integrates the repository (deletes its repository context); it does not touch the project on GitLab. " +
			"Import with `installation_id:gitlab_group_id:gitlab_project_id`.",
		Attributes: attrs,
	}
}

func (r *gitlabRepositoryResource) ops(plan *gitlabRepositoryModel) repoOps {
	installationID := plan.InstallationID.ValueString()
	projectID := plan.GitlabProjectID.ValueInt64()
	return repoOps{
		scmName: "GitLab",
		integrate: func() error {
			return r.apiClient.IntegrateGitlabRepository(api_client.GitlabRepositoryIntegrate{
				InstallationID:  installationID,
				GitlabGroupID:   plan.GitlabGroupID.ValueInt64(),
				GitlabProjectID: projectID,
				Name:            plan.Name.ValueString(),
				URL:             plan.URL.ValueString(),
				Branch:          plan.Branch.ValueString(),
				ProjectID:       plan.ProjectID.ValueString(),
			})
		},
		find: func() (*api_client.ScmRepository, error) {
			return r.apiClient.FindGitlabRepository(installationID, projectID)
		},
		update: r.apiClient.UpdateGitlabRepositories,
	}
}

func (r *gitlabRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gitlabRepositoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	row := createRepo(r.ops(&plan), &plan.RepoConfigFields, &resp.Diagnostics)
	if row == nil {
		return
	}
	plan.RepoConfigFields = fromAPI(plan.RepoConfigFields, row)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gitlabRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gitlabRepositoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	row, err := r.apiClient.FindGitlabRepository(state.InstallationID.ValueString(), state.GitlabProjectID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitLab repository integration", err.Error())
		return
	}
	if row == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.RepoConfigFields = fromAPI(state.RepoConfigFields, row)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *gitlabRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state gitlabRepositoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	row := updateRepo(r.apiClient, r.ops(&plan), &plan.RepoConfigFields, &state.RepoConfigFields, &resp.Diagnostics)
	if row == nil {
		return
	}
	plan.RepoConfigFields = fromAPI(plan.RepoConfigFields, row)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gitlabRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gitlabRepositoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteRepo(r.apiClient, r.ops(&state), &state.RepoConfigFields, &resp.Diagnostics)
}

func (r *gitlabRepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "expected format installation_id:gitlab_group_id:gitlab_project_id")
		return
	}
	groupID, err1 := strconv.ParseInt(parts[1], 10, 64)
	projectID, err2 := strconv.ParseInt(parts[2], 10, 64)
	if err1 != nil || err2 != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("gitlab ids must be numeric: %v %v", err1, err2))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("installation_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("gitlab_group_id"), groupID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("gitlab_project_id"), projectID)...)
}
