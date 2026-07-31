package shift_left_repository

import (
	"context"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type azureRepositoryModel struct {
	InstallationID    types.String `tfsdk:"installation_id"`
	AccountName       types.String `tfsdk:"account_name"`
	AzureRepositoryID types.String `tfsdk:"azure_repository_id"`
	AzureProjectID    types.String `tfsdk:"azure_project_id"`
	RepoConfigFields
}

func NewAzureDevopsRepositoryResource() resource.Resource {
	return NewRepoResource(RepoSpec[azureRepositoryModel]{
		TypeNameSuffix: "_shift_left_azure_devops_repository",
		SchemaFn:       azureRepositorySchema,
		ImportFn:       azureRepositoryImportState,
		Ops:            azureRepositoryOps,
		Fields:         (*azureRepositoryModel).Fields,
	})
}

func azureRepositorySchema() rschema.Schema {
	attrs := sharedRepoAttributes(azureTraits)
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
		Description:   "Azure DevOps repository UUID (from Azure DevOps, not minted by Orca).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["azure_project_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Azure DevOps project UUID containing the repository (from Azure DevOps, not minted by Orca).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	return rschema.Schema{
		Description: "Integrates a single Azure DevOps repository into Orca Shift Left under an existing Azure DevOps installation. " +
			"`azure_project_id` and `azure_repository_id` are Azure DevOps-side identifiers (obtain them from Azure DevOps; " +
			"`orcasecurity_shift_left_azure_devops_repositories` returns the repository id but not the project id). " +
			"Destroying the resource un-integrates the repository (deletes its repository context); it does not touch the repository on Azure DevOps. " +
			"Import with `installation_id:account_name:azure_project_id:azure_repository_id`.",
		Attributes: attrs,
	}
}

func azureRepositoryOps(apiClient *api_client.APIClient, plan *azureRepositoryModel) repoOps {
	installationID := plan.InstallationID.ValueString()
	accountName := plan.AccountName.ValueString()
	repoID := plan.AzureRepositoryID.ValueString()
	return repoOps{
		client: apiClient,
		traits: azureTraits,
		integrate: func() error {
			return apiClient.IntegrateAzureRepository(api_client.AzureRepositoryIntegrate{
				InstallationID:    plan.InstallationID.ValueString(),
				AccountName:       accountName,
				AzureRepositoryID: repoID,
				AzureProjectID:    plan.AzureProjectID.ValueString(),
				Name:              plan.Name.ValueString(),
				URL:               plan.URL.ValueString(),
				Branch:            plan.Branch.ValueString(),
				ProjectID:         plan.ProjectID.ValueString(),
				Config:            integrateConfig(&plan.RepoConfigFields),
			})
		},
		find: func() (*api_client.ScmRepository, error) {
			return apiClient.FindAzureRepository(installationID, accountName, repoID)
		},
		update: apiClient.UpdateAzureRepositories,
	}
}

func azureRepositoryImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
