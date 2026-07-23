package shift_left_gitlab_group

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                = &gitlabGroupResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupResource{}
	_ resource.ResourceWithImportState = &gitlabGroupResource{}
)

var gitlabLabels = shift_left_integration.AdoptLabels{
	NotFoundTitle:  "GitLab group not found",
	NilReadTitle:   "Error reading gitlab group after write",
	NilReadDetail:  "The gitlab group was configured but could not be read back; the API may not have propagated the change yet. Re-run terraform apply.",
	ReadErrorTitle: "Error reading GitLab group",
	DeleteLog:      "Removing GitLab group from state; the live integration is left untouched.",
	MissingWarn:    "GitLab group %s missing remotely",
}

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

func (r *gitlabGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	installationID := plan.InstallationID.ValueString()
	groupID := plan.GroupID.ValueString()
	grp := r.adopt(&resp.Diagnostics, installationID, groupID, plan, config,
		fmt.Sprintf("Group %q on installation %q does not exist. Integrate the Orca GitLab group first, then import.", groupID, installationID),
		"Error configuring GitLab group")
	if grp == nil {
		return
	}
	state := apiToState(grp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *gitlabGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	installationID := state.InstallationID.ValueString()
	groupID := state.GroupID.ValueString()
	grp := shift_left_integration.ReadUnit(ctx, &resp.Diagnostics, gitlabLabels, groupID,
		func() (*api_client.GitlabGroup, error) {
			return r.apiClient.GetGitlabGroup(installationID, groupID)
		},
		resp.State.RemoveResource,
	)
	if grp == nil {
		return
	}
	newState := apiToState(grp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *gitlabGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	installationID := plan.InstallationID.ValueString()
	groupID := plan.GroupID.ValueString()
	grp := r.adopt(&resp.Diagnostics, installationID, groupID, plan, config,
		fmt.Sprintf("Group %q on installation %q was not found. It may have been removed; re-import.", groupID, installationID),
		"Error updating GitLab group")
	if grp == nil {
		return
	}
	state := apiToState(grp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *gitlabGroupResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	shift_left_integration.DeleteNoop(ctx, gitlabLabels)
}

func (r *gitlabGroupResource) adopt(
	diags *diag.Diagnostics, installationID, groupID string, plan, config resourceModel, notFoundMsg, writeTitle string,
) *api_client.GitlabGroup {
	return shift_left_integration.AdoptWrite(
		diags, gitlabLabels, notFoundMsg, writeTitle,
		func() (*api_client.GitlabGroup, error) {
			return r.apiClient.GetGitlabGroup(installationID, groupID)
		},
		func(body api_client.ScmInstallationUpdate) (*api_client.GitlabGroup, error) {
			return r.apiClient.UpdateGitlabGroup(installationID, groupID, body)
		},
		func(u *api_client.GitlabGroup) shift_left_integration.ExistingUnit {
			return shift_left_integration.ExistingFromAPI(u.InstallationMode, u.DefaultPolicies, u.Policies, u.Project, u.ConfigSettings)
		},
		plan.InstallationMode, plan.DefaultPolicies, plan.PoliciesIds, plan.ConfigSettings,
		shift_left_integration.ProjectIntentFrom(config.ProjectID, config.PoliciesIds, config.DefaultPolicies),
	)
}
