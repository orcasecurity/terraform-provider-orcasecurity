package shift_left_github_installation

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                = &githubInstallationResource{}
	_ resource.ResourceWithConfigure   = &githubInstallationResource{}
	_ resource.ResourceWithImportState = &githubInstallationResource{}
)

var githubLabels = shift_left_integration.AdoptLabels{
	NotFoundTitle:  "GitHub installation not found",
	NilReadTitle:   "Error reading github installation after write",
	NilReadDetail:  "The github installation was configured but could not be read back; the API may not have propagated the change yet. Re-run terraform apply.",
	ReadErrorTitle: "Error reading GitHub installation",
	DeleteLog:      "Removing GitHub installation from state; the live integration is left untouched.",
	MissingWarn:    "GitHub installation %s missing remotely",
}

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

func (r *githubInstallationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := plan.InstallationID.ValueString()
	inst := r.adopt(&resp.Diagnostics, id, plan, config,
		fmt.Sprintf("Installation %q does not exist. Install the Orca GitHub App first, then import.", id),
		"Error configuring GitHub installation")
	if inst == nil {
		return
	}
	state := apiToState(inst)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *githubInstallationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.InstallationID.ValueString()
	inst := shift_left_integration.ReadUnit(ctx, &resp.Diagnostics, githubLabels, id,
		func() (*api_client.GithubInstallation, error) { return r.apiClient.GetGithubInstallation(id) },
		resp.State.RemoveResource,
	)
	if inst == nil {
		return
	}
	newState := apiToState(inst)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *githubInstallationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := plan.InstallationID.ValueString()
	inst := r.adopt(&resp.Diagnostics, id, plan, config,
		fmt.Sprintf("Installation %q was not found. It may have been removed; re-import.", id),
		"Error updating GitHub installation")
	if inst == nil {
		return
	}
	state := apiToState(inst)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *githubInstallationResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	shift_left_integration.DeleteNoop(ctx, githubLabels)
}

func (r *githubInstallationResource) adopt(
	diags *diag.Diagnostics, id string, plan, config resourceModel, notFoundMsg, writeTitle string,
) *api_client.GithubInstallation {
	return shift_left_integration.AdoptWrite(
		diags, githubLabels, notFoundMsg, writeTitle,
		func() (*api_client.GithubInstallation, error) { return r.apiClient.GetGithubInstallation(id) },
		func(body api_client.ScmInstallationUpdate) (*api_client.GithubInstallation, error) {
			return r.apiClient.UpdateGithubInstallation(id, body)
		},
		func(u *api_client.GithubInstallation) shift_left_integration.ExistingUnit {
			return shift_left_integration.ExistingFromAPI(u.InstallationMode, u.DefaultPolicies, u.Policies, u.Project, u.ConfigSettings)
		},
		plan.InstallationMode, plan.DefaultPolicies, plan.PoliciesIds, plan.ConfigSettings,
		shift_left_integration.ProjectIntentFrom(config.ProjectID, config.PoliciesIds, config.DefaultPolicies),
	)
}
