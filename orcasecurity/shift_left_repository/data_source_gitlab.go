package shift_left_repository

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var gitlabRepositoriesSpec = shift_left_integration.ScmObjectListSpec[api_client.GitlabRepositoryListItem]{
	TypeNameSuffix: "_shift_left_gitlab_repositories",
	Description: "Lists all Orca GitLab shift-left integrated repositories for fleet-wide for_each. " +
		"`installation_id` is the Orca GitLab installation UUID.",
	CollectionKey:  "repositories",
	ListErrorTitle: "Error listing GitLab repositories",
	Attrs: mergeAttrs(sharedRepoListAttrs(), map[string]dschema.Attribute{
		"installation_id":   dschema.StringAttribute{Computed: true, Description: "Orca GitLab installation UUID."},
		"gitlab_project_id": dschema.Int64Attribute{Computed: true},
	}),
	AttrTypes: mergeTypes(sharedRepoListAttrTypes(), map[string]attr.Type{
		"installation_id":   types.StringType,
		"gitlab_project_id": types.Int64Type,
	}),
	List: func(c *api_client.APIClient) ([]api_client.GitlabRepositoryListItem, error) {
		return c.ListGitlabRepositories()
	},
	Row: func(a *api_client.GitlabRepositoryListItem) map[string]attr.Value {
		return mergeValues(sharedRepoListValues(a.ScmRepository), map[string]attr.Value{
			"installation_id":   types.StringValue(a.InstallationID),
			"gitlab_project_id": types.Int64Value(a.GitlabProjectID),
		})
	},
}

func NewGitlabRepositoriesDataSource() datasource.DataSource {
	return shift_left_integration.NewScmObjectListDataSource(gitlabRepositoriesSpec)
}

func gitlabRepositoriesToListValue(rows []api_client.GitlabRepositoryListItem) (types.List, diag.Diagnostics) {
	return gitlabRepositoriesSpec.ListValue(rows)
}
