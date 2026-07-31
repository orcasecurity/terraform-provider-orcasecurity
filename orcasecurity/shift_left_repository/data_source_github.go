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

var githubRepositoriesSpec = shift_left_integration.ScmObjectListSpec[api_client.GithubRepositoryListItem]{
	TypeNameSuffix: "_shift_left_github_repositories",
	Description: "Lists all Orca GitHub shift-left integrated repositories for fleet-wide for_each. " +
		"`account_id` is the Orca GitHub account UUID (same identity as `orcasecurity_shift_left_github_account.account_id`).",
	CollectionKey:  "repositories",
	ListErrorTitle: "Error listing GitHub repositories",
	Attrs: mergeAttrs(sharedRepoListAttrs(), map[string]dschema.Attribute{
		"account_id": dschema.StringAttribute{
			Computed:    true,
			Description: "Orca UUID of the GitHub integrated account owning the repository.",
		},
		"github_repository_id": dschema.Int64Attribute{Computed: true},
	}),
	AttrTypes: mergeTypes(sharedRepoListAttrTypes(), map[string]attr.Type{
		"account_id":           types.StringType,
		"github_repository_id": types.Int64Type,
	}),
	List: func(c *api_client.APIClient) ([]api_client.GithubRepositoryListItem, error) {
		return c.ListGithubRepositories()
	},
	Row: func(a *api_client.GithubRepositoryListItem) map[string]attr.Value {
		return mergeValues(sharedRepoListValues(a.ScmRepository), map[string]attr.Value{
			"account_id":           types.StringValue(a.AccountID),
			"github_repository_id": types.Int64Value(a.GithubRepositoryID),
		})
	},
}

func NewGithubRepositoriesDataSource() datasource.DataSource {
	return shift_left_integration.NewScmObjectListDataSource(githubRepositoriesSpec)
}

func githubRepositoriesToListValue(rows []api_client.GithubRepositoryListItem) (types.List, diag.Diagnostics) {
	return githubRepositoriesSpec.ListValue(rows)
}
