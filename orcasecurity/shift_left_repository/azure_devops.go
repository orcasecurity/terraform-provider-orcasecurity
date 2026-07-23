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
	_ resource.Resource                = &azureRepositoryResource{}
	_ resource.ResourceWithConfigure   = &azureRepositoryResource{}
	_ resource.ResourceWithImportState = &azureRepositoryResource{}
)

type azureRepositoryResource struct {
	apiClient *api_client.APIClient
}

func NewAzureDevopsRepositoryResource() resource.Resource { return &azureRepositoryResource{} }

type azureRepositoryModel struct {
	InstallationID    types.String `tfsdk:"installation_id"`
	AccountName       types.String `tfsdk:"account_name"`
	AzureRepositoryID types.String `tfsdk:"azure_repository_id"`
	AzureProjectID    types.String `tfsdk:"azure_project_id"`
	RepoConfigFields
}

func (r *azureRepositoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_azure_devops_repository"
}

func (r *azureRepositoryResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.apiClient = shift_left_integration.ConfigureAPIClient(req)
}

func (r *azureRepositoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := sharedRepoAttributes("Azure DevOps", fullSkipCheckRuns)
	attrs["installation_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca id of the Azure DevOps installation (see `orcasecurity_shift_left_azure_devops_installation`).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["account_name"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Azure DevOps organization name owning the repository.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["azure_repository_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Azure DevOps repository UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["azure_project_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Azure DevOps project UUID containing the repository.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	resp.Schema = rschema.Schema{
		Description: "Integrates a single Azure DevOps repository into Orca Shift Left under an existing Azure DevOps installation. " +
			"Destroying the resource un-integrates the repository (deletes its repository context); it does not touch the repository on Azure DevOps. " +
			"Import with `installation_id:account_name:azure_project_id:azure_repository_id`.",
		Attributes: attrs,
	}
}

func (r *azureRepositoryResource) ops(plan *azureRepositoryModel) repoOps {
	accountName := plan.AccountName.ValueString()
	repoID := plan.AzureRepositoryID.ValueString()
	return repoOps{
		scmName: "Azure DevOps",
		integrate: func() error {
			return r.apiClient.IntegrateAzureRepository(api_client.AzureRepositoryIntegrate{
				InstallationID:    plan.InstallationID.ValueString(),
				AccountName:       accountName,
				AzureRepositoryID: repoID,
				AzureProjectID:    plan.AzureProjectID.ValueString(),
				Name:              plan.Name.ValueString(),
				URL:               plan.URL.ValueString(),
				Branch:            plan.Branch.ValueString(),
				ProjectID:         plan.ProjectID.ValueString(),
			})
		},
		find: func() (*api_client.ScmRepository, error) {
			return r.apiClient.FindAzureRepository(accountName, repoID)
		},
		update: r.apiClient.UpdateAzureRepositories,
	}
}

func (r *azureRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan azureRepositoryModel
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

func (r *azureRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state azureRepositoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	row, err := r.apiClient.FindAzureRepository(state.AccountName.ValueString(), state.AzureRepositoryID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Azure DevOps repository integration", err.Error())
		return
	}
	if row == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.RepoConfigFields = fromAPI(state.RepoConfigFields, row)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *azureRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state azureRepositoryModel
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

func (r *azureRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state azureRepositoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteRepo(r.apiClient, r.ops(&state), &state.RepoConfigFields, &resp.Diagnostics)
}

func (r *azureRepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid import ID", "expected format installation_id:account_name:azure_project_id:azure_repository_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("installation_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("azure_project_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("azure_repository_id"), parts[3])...)
}
