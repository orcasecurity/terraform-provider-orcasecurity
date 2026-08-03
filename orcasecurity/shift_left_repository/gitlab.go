package shift_left_repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type gitlabRepositoryModel struct {
	InstallationID  types.String `tfsdk:"installation_id"`
	GitlabGroupID   types.Int64  `tfsdk:"gitlab_group_id"`
	GitlabProjectID types.Int64  `tfsdk:"gitlab_project_id"`
	RepoConfigFields
}

func NewGitlabRepositoryResource() resource.Resource {
	return NewRepoResource(RepoSpec[gitlabRepositoryModel]{
		TypeNameSuffix: "_shift_left_gitlab_repository",
		SchemaFn:       gitlabRepositorySchema,
		ImportFn:       gitlabRepositoryImportState,
		Ops:            gitlabRepositoryOps,
	})
}

func gitlabRepositorySchema() rschema.Schema {
	attrs := sharedRepoAttributes(gitlabTraits)
	attrs["installation_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca id of the GitLab installation (see `orcasecurity_shift_left_gitlab_installation`).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["gitlab_group_id"] = rschema.Int64Attribute{
		Required: true,
		Description: "Numeric GitLab group id owning the project (from GitLab, not minted by Orca). " +
			"If the group is not yet integrated with Orca, integrating the first repository also creates the group unit.",
		PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
	}
	attrs["gitlab_project_id"] = rschema.Int64Attribute{
		Required:      true,
		Description:   "Numeric GitLab project (repository) id (from GitLab, not minted by Orca).",
		PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
	}
	return rschema.Schema{
		Description: "Integrates a single GitLab project (repository) into Orca Shift Left under an existing GitLab installation. " +
			"`gitlab_group_id` and `gitlab_project_id` are GitLab-side identifiers (obtain them from GitLab or " +
			"`orcasecurity_shift_left_gitlab_repositories`). " +
			"Destroying the resource un-integrates the repository (deletes its repository context); it does not touch the project on GitLab. " +
			"Import with `installation_id:gitlab_group_id:gitlab_project_id`.",
		Attributes: attrs,
	}
}

func gitlabRepositoryOps(apiClient *api_client.APIClient, plan *gitlabRepositoryModel) repoOps {
	installationID := plan.InstallationID.ValueString()
	projectID := plan.GitlabProjectID.ValueInt64()
	return repoOps{
		client: apiClient,
		traits: gitlabTraits,
		fields: &plan.RepoConfigFields,
		integrate: func() error {
			cfg, _ := planRepoConfig(&plan.RepoConfigFields)
			return apiClient.IntegrateGitlabRepository(api_client.GitlabRepositoryIntegrate{
				InstallationID:  installationID,
				GitlabGroupID:   plan.GitlabGroupID.ValueInt64(),
				GitlabProjectID: projectID,
				Name:            plan.Name.ValueString(),
				URL:             plan.URL.ValueString(),
				Branch:          plan.Branch.ValueString(),
				ProjectID:       plan.ProjectID.ValueString(),
				Config:          cfg,
			})
		},
		find: func() (*api_client.ScmRepository, error) {
			return apiClient.FindGitlabRepository(installationID, projectID)
		},
		update: apiClient.UpdateGitlabRepositories,
	}
}

func gitlabRepositoryImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
