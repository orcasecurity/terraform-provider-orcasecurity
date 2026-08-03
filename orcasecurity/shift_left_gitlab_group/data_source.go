package shift_left_gitlab_group

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var groupsSpec = shift_left_common.ScmObjectListSpec[api_client.GitlabGroup]{
	TypeNameSuffix: "_shift_left_gitlab_groups",
	Description:    "Lists all Orca GitLab shift-left integrated groups for fleet-wide for_each. Use gitlab_group_id with orcasecurity_shift_left_gitlab_group.",
	CollectionKey:  "groups",
	ListErrorTitle: "Error listing GitLab groups",
	Attrs: shift_left_common.MergeMaps(shift_left_integration.SharedScmListUnitAttrs(), map[string]dschema.Attribute{
		"id":              dschema.StringAttribute{Computed: true, Description: "Orca UUID of the GitLab group unit."},
		"installation_id": dschema.StringAttribute{Computed: true, Description: "Orca UUID of the parent GitLab installation."},
		"gitlab_group_id": dschema.Int64Attribute{
			Computed:    true,
			Description: "Numeric GitLab group id (from GitLab, not minted by Orca).",
		},
	}),
	AttrTypes: shift_left_common.MergeMaps(shift_left_integration.SharedScmListUnitAttrTypes(), map[string]attr.Type{
		"id":              types.StringType,
		"installation_id": types.StringType,
		"gitlab_group_id": types.Int64Type,
	}),
	List: func(c *api_client.APIClient) ([]api_client.GitlabGroup, error) {
		return c.ListGitlabGroups()
	},
	Row: func(g *api_client.GitlabGroup) map[string]attr.Value {
		return shift_left_common.MergeMaps(
			shift_left_integration.SharedScmListUnitValues(g.AccountName, g.ScmUnitCommonFields),
			map[string]attr.Value{
				"id":              types.StringValue(g.ID),
				"installation_id": types.StringValue(g.InstallationID),
				"gitlab_group_id": types.Int64Value(g.GitlabGroupID),
			})
	},
}

func NewGroupsDataSource() datasource.DataSource {
	return shift_left_common.NewScmObjectListDataSource(groupsSpec)
}

func groupsToListValue(grps []api_client.GitlabGroup) (types.List, diag.Diagnostics) {
	return groupsSpec.ListValue(grps)
}
