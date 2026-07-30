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

type githubRepositoryModel struct {
	InstallationID     types.String `tfsdk:"installation_id"`
	GithubRepositoryID types.Int64  `tfsdk:"github_repository_id"`
	RepoConfigFields
}

func NewGithubRepositoryResource() resource.Resource {
	return NewRepoResource(RepoSpec[githubRepositoryModel]{
		TypeNameSuffix: "_shift_left_github_repository",
		SchemaFn:       githubRepositorySchema,
		ImportFn:       githubRepositoryImportState,
		Ops:            githubRepositoryOps,
		Fields:         (*githubRepositoryModel).Fields,
	})
}

func githubRepositorySchema() rschema.Schema {
	attrs := sharedRepoAttributes(githubTraits)
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
	return rschema.Schema{
		Description: "Integrates a single GitHub repository into Orca Shift Left under an existing GitHub installation. " +
			"Destroying the resource un-integrates the repository (deletes its repository context); it does not touch the repository on GitHub. " +
			"Import with `installation_id:github_repository_id`.",
		Attributes: attrs,
	}
}

func githubRepositoryOps(apiClient *api_client.APIClient, plan *githubRepositoryModel) repoOps {
	installationID := plan.InstallationID.ValueString()
	githubRepositoryID := plan.GithubRepositoryID.ValueInt64()
	return repoOps{
		client: apiClient,
		traits: githubTraits,
		integrate: func() error {
			return apiClient.IntegrateGithubRepository(api_client.GithubRepositoryIntegrate{
				InstallationID:     installationID,
				GithubRepositoryID: githubRepositoryID,
				Name:               plan.Name.ValueString(),
				URL:                plan.URL.ValueString(),
				Branch:             plan.Branch.ValueString(),
				ProjectID:          plan.ProjectID.ValueString(),
				Config:             integrateConfig(&plan.RepoConfigFields),
			})
		},
		find: func() (*api_client.ScmRepository, error) {
			return apiClient.FindGithubRepository(installationID, plan.Name.ValueString(), githubRepositoryID)
		},
		update: apiClient.UpdateGithubRepositories,
	}
}

func githubRepositoryImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "expected format installation_id:github_repository_id")
		return
	}
	numericID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("github_repository_id must be numeric: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("installation_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("github_repository_id"), numericID)...)
}
