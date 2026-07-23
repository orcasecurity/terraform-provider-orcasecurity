package shift_left_repository

import (
	"context"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &bitbucketRepositoryResource{}
	_ resource.ResourceWithConfigure   = &bitbucketRepositoryResource{}
	_ resource.ResourceWithImportState = &bitbucketRepositoryResource{}
)

type bitbucketRepositoryResource struct {
	apiClient *api_client.APIClient
}

func NewBitbucketRepositoryResource() resource.Resource { return &bitbucketRepositoryResource{} }

type bitbucketRepositoryModel struct {
	InstallationID        types.String `tfsdk:"installation_id"`
	AccountID             types.String `tfsdk:"account_id"`
	BitbucketRepositoryID types.String `tfsdk:"bitbucket_repository_id"`
	Slug                  types.String `tfsdk:"slug"`
	RepoConfigFields
}

func (r *bitbucketRepositoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_bitbucket_repository"
}

func (r *bitbucketRepositoryResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.apiClient = shift_left_integration.ConfigureAPIClient(req)
}

func (r *bitbucketRepositoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := sharedRepoAttributes("Bitbucket", fullSkipCheckRuns)
	attrs["installation_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca id of the Bitbucket installation (see `orcasecurity_shift_left_bitbucket_installation`).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["account_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Bitbucket workspace slug (cloud) or project key (server) owning the repository.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["bitbucket_repository_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Bitbucket repository id.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["slug"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Bitbucket repository slug.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	resp.Schema = rschema.Schema{
		Description: "Integrates a single Bitbucket repository into Orca Shift Left under an existing Bitbucket installation. " +
			"Destroying the resource un-integrates the repository (deletes its repository context); it does not touch the repository on Bitbucket. " +
			"Import with `installation_id:account_id:bitbucket_repository_id`.",
		Attributes: attrs,
	}
}

func (r *bitbucketRepositoryResource) ops(plan *bitbucketRepositoryModel) repoOps {
	accountID := plan.AccountID.ValueString()
	repoID := plan.BitbucketRepositoryID.ValueString()
	return repoOps{
		scmName: "Bitbucket",
		integrate: func() error {
			return r.apiClient.IntegrateBitbucketRepository(api_client.BitbucketRepositoryIntegrate{
				InstallationID:        plan.InstallationID.ValueString(),
				AccountID:             accountID,
				BitbucketRepositoryID: repoID,
				Slug:                  plan.Slug.ValueString(),
				Name:                  plan.Name.ValueString(),
				URL:                   plan.URL.ValueString(),
				Branch:                plan.Branch.ValueString(),
				ProjectID:             plan.ProjectID.ValueString(),
			})
		},
		find: func() (*api_client.ScmRepository, error) {
			return r.apiClient.FindBitbucketRepository(accountID, repoID)
		},
		update: r.apiClient.UpdateBitbucketRepositories,
	}
}

func (r *bitbucketRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bitbucketRepositoryModel
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

func (r *bitbucketRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bitbucketRepositoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	row, err := r.apiClient.FindBitbucketRepository(state.AccountID.ValueString(), state.BitbucketRepositoryID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Bitbucket repository integration", err.Error())
		return
	}
	if row == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.RepoConfigFields = fromAPI(state.RepoConfigFields, row)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *bitbucketRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state bitbucketRepositoryModel
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

func (r *bitbucketRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bitbucketRepositoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteRepo(r.apiClient, r.ops(&state), &state.RepoConfigFields, &resp.Diagnostics)
}

func (r *bitbucketRepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "expected format installation_id:account_id:bitbucket_repository_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("installation_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bitbucket_repository_id"), parts[2])...)
}
