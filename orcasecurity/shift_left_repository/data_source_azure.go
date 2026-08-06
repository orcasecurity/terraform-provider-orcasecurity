package shift_left_repository

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var azureRepositoriesSpec = shift_left_common.ScmObjectListSpec[api_client.AzureRepositoryListItem]{
	TypeNameSuffix: "_shift_left_azure_devops_repositories",
	Description: "Lists all Orca Azure DevOps shift-left integrated repositories for fleet-wide for_each. " +
		"`installation_id` is joined from the owning integrated account, and `azure_project_id` is joined from " +
		"the per-account Azure DevOps repository browse endpoint (the integrated repositories list omits it), " +
		"so each row round-trips directly into `orcasecurity_shift_left_azure_devops_repository`.",
	CollectionKey:  "repositories",
	ListErrorTitle: "Error listing Azure DevOps repositories",
	Attrs: shift_left_common.MergeMaps(sharedRepoListAttrs(), map[string]dschema.Attribute{
		"installation_id":     dschema.StringAttribute{Computed: true, Description: "Orca Azure DevOps installation UUID."},
		"account_name":        dschema.StringAttribute{Computed: true, Description: "Azure DevOps organization name."},
		"azure_repository_id": dschema.StringAttribute{Computed: true, Description: "Azure DevOps repository UUID (from Azure DevOps)."},
		"azure_project_id":    dschema.StringAttribute{Computed: true, Description: "Azure DevOps project UUID containing the repository (from Azure DevOps)."},
	}),
	AttrTypes: shift_left_common.MergeMaps(sharedRepoListAttrTypes(), map[string]attr.Type{
		"installation_id":     types.StringType,
		"account_name":        types.StringType,
		"azure_repository_id": types.StringType,
		"azure_project_id":    types.StringType,
	}),
	List: func(c *api_client.APIClient) ([]api_client.AzureRepositoryListItem, error) {
		return c.ListAzureRepositories()
	},
	Row: func(a *api_client.AzureRepositoryListItem) map[string]attr.Value {
		return shift_left_common.MergeMaps(sharedRepoListValues(a.ScmRepository), map[string]attr.Value{
			"installation_id":     tfconv.StringOrNull(a.InstallationID),
			"account_name":        types.StringValue(a.AccountName),
			"azure_repository_id": types.StringValue(a.AzureRepositoryID),
			"azure_project_id":    tfconv.StringOrNull(a.AzureProjectID),
		})
	},
}

func NewAzureDevopsRepositoriesDataSource() datasource.DataSource {
	return shift_left_common.NewScmObjectListDataSource(azureRepositoriesSpec)
}

func azureRepositoriesToListValue(rows []api_client.AzureRepositoryListItem) (types.List, diag.Diagnostics) {
	return azureRepositoriesSpec.ListValue(rows)
}
