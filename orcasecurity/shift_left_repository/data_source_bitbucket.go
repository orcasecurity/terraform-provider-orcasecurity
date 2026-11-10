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

var bitbucketRepositoriesSpec = shift_left_common.ScmObjectListSpec[api_client.BitbucketRepositoryListItem]{
	TypeNameSuffix: "_shift_left_bitbucket_repositories",
	Description: "Lists all Orca Bitbucket shift-left integrated repositories for fleet-wide for_each. " +
		"`account_id` is the Bitbucket workspace slug (cloud) or project key (server) — not an Orca UUID. " +
		"`installation_id` is joined from the owning integrated account.",
	CollectionKey:  "repositories",
	ListErrorTitle: "Error listing Bitbucket repositories",
	Attrs: shift_left_common.MergeMaps(sharedRepoListAttrs(), map[string]dschema.Attribute{
		"installation_id": dschema.StringAttribute{Computed: true, Description: "Orca Bitbucket installation UUID."},
		"account_id": dschema.StringAttribute{
			Computed:    true,
			Description: "Bitbucket workspace slug (cloud) or project key (server); not an Orca UUID.",
		},
		"bitbucket_repository_id": dschema.StringAttribute{
			Computed:    true,
			Description: "Bitbucket repository id (from Bitbucket, not minted by Orca).",
		},
		"slug": dschema.StringAttribute{
			Computed:    true,
			Description: "Bitbucket repository slug (from Bitbucket).",
		},
	}),
	AttrTypes: shift_left_common.MergeMaps(sharedRepoListAttrTypes(), map[string]attr.Type{
		"installation_id":         types.StringType,
		"account_id":              types.StringType,
		"bitbucket_repository_id": types.StringType,
		"slug":                    types.StringType,
	}),
	List: func(c *api_client.APIClient) ([]api_client.BitbucketRepositoryListItem, error) {
		return c.ListBitbucketRepositories()
	},
	Row: func(a *api_client.BitbucketRepositoryListItem) map[string]attr.Value {
		return shift_left_common.MergeMaps(sharedRepoListValues(a.ScmRepository), map[string]attr.Value{
			"installation_id":         tfconv.StringOrNull(a.InstallationID),
			"account_id":              types.StringValue(a.AccountID),
			"bitbucket_repository_id": types.StringValue(a.BitbucketRepositoryID),
			"slug":                    tfconv.StringOrNull(a.Slug),
		})
	},
}

func NewBitbucketRepositoriesDataSource() datasource.DataSource {
	return shift_left_common.NewScmObjectListDataSource(bitbucketRepositoriesSpec)
}

func bitbucketRepositoriesToListValue(rows []api_client.BitbucketRepositoryListItem) (types.List, diag.Diagnostics) {
	return bitbucketRepositoriesSpec.ListValue(rows)
}
