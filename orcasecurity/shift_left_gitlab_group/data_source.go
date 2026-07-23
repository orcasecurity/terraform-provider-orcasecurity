package shift_left_gitlab_group

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var groupsSpec = shift_left_integration.ScmUnitListSpec[api_client.GitlabGroup]{
	TypeNameSuffix: "_shift_left_gitlab_groups",
	Description:    "Lists all Orca GitLab shift-left integrated groups for fleet-wide for_each.",
	CollectionKey:  "groups",
	ListErrorTitle: "Error listing GitLab groups",
	Extra: map[string]attr.Type{
		"id":              types.StringType,
		"installation_id": types.StringType,
		"group_id":        types.StringType,
		"gitlab_group_id": types.Int64Type,
	},
	List: func(c *api_client.APIClient) ([]api_client.GitlabGroup, error) {
		return c.ListGitlabGroups()
	},
	Row: func(g *api_client.GitlabGroup) (string, api_client.ScmUnitCommonFields, map[string]attr.Value) {
		return g.AccountName, g.ScmUnitCommonFields, map[string]attr.Value{
			"id":              types.StringValue(g.ID),
			"installation_id": types.StringValue(g.InstallationID),
			"group_id":        types.StringValue(g.ID),
			"gitlab_group_id": types.Int64Value(g.GitlabGroupID),
		}
	},
}

func NewGroupsDataSource() datasource.DataSource {
	return shift_left_integration.NewScmUnitListDataSource(groupsSpec)
}

func groupsToListValue(grps []api_client.GitlabGroup) (types.List, diag.Diagnostics) {
	return groupsSpec.ListValue(grps)
}
