package shift_left_gitlab_group

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &gitlabGroupResource{}
	_ resource.ResourceWithConfigure   = &gitlabGroupResource{}
	_ resource.ResourceWithImportState = &gitlabGroupResource{}
)

type gitlabGroupResource struct {
	apiClient *api_client.APIClient
}

func NewResource() resource.Resource { return &gitlabGroupResource{} }

func (r *gitlabGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_gitlab_group"
}

func (r *gitlabGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *gitlabGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceSchema()
}

func (r *gitlabGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", "expected <installation_id>/<group_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("installation_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), parts[1])...)
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
	grp, err := r.write(installationID, groupID, plan, config)
	if errors.Is(err, shift_left_integration.ErrUnitNotFound) {
		resp.Diagnostics.AddError("GitLab group not found",
			fmt.Sprintf("Group %q on installation %q does not exist. Integrate the Orca GitLab group first, then import.", groupID, installationID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error configuring GitLab group", err.Error())
		return
	}
	if grp == nil {
		resp.Diagnostics.AddError(
			"Error reading gitlab group after write",
			"The gitlab group was configured but could not be read back; the API may not have propagated the change yet. Re-run terraform apply.",
		)
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
	grp, err := r.apiClient.GetGitlabGroup(state.InstallationID.ValueString(), state.GroupID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitLab group", err.Error())
		return
	}
	if grp == nil {
		tflog.Warn(ctx, fmt.Sprintf("GitLab group %s missing remotely", state.GroupID.ValueString()))
		resp.State.RemoveResource(ctx)
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
	grp, err := r.write(installationID, groupID, plan, config)
	if errors.Is(err, shift_left_integration.ErrUnitNotFound) {
		resp.Diagnostics.AddError("GitLab group not found",
			fmt.Sprintf("Group %q on installation %q was not found. It may have been removed; re-import.", groupID, installationID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error updating GitLab group", err.Error())
		return
	}
	if grp == nil {
		resp.Diagnostics.AddError(
			"Error reading gitlab group after write",
			"The gitlab group was configured but could not be read back; the API may not have propagated the change yet. Re-run terraform apply.",
		)
		return
	}
	state := apiToState(grp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state only. The GitLab integrated group is
// not owned by Terraform (created by integrating the Orca GitLab app) and is
// left untouched.
func (r *gitlabGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Removing GitLab group from state; the live integration is left untouched.")
}

func (r *gitlabGroupResource) write(installationID, groupID string, plan, config resourceModel) (*api_client.GitlabGroup, error) {
	project := shift_left_integration.ProjectIntentFrom(config.ProjectID, config.PoliciesIds, config.DefaultPolicies)
	return shift_left_integration.WriteAdopted(
		func() (*api_client.GitlabGroup, error) {
			return r.apiClient.GetGitlabGroup(installationID, groupID)
		},
		func(body api_client.ScmInstallationUpdate) (*api_client.GitlabGroup, error) {
			return r.apiClient.UpdateGitlabGroup(installationID, groupID, body)
		},
		func(u *api_client.GitlabGroup) shift_left_integration.ExistingUnit {
			return shift_left_integration.ExistingFromAPI(u.InstallationMode, u.DefaultPolicies, u.Policies, u.Project, u.ConfigSettings)
		},
		plan.InstallationMode, plan.DefaultPolicies, plan.PoliciesIds, plan.ConfigSettings, project,
	)
}
