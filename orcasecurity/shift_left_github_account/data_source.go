package shift_left_github_account

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var accountsSpec = shift_left_common.ScmObjectListSpec[api_client.GithubInstallation]{
	TypeNameSuffix: "_shift_left_github_accounts",
	Description: "Lists all Orca GitHub shift-left integrated accounts for fleet-wide for_each. " +
		"account_id is the Orca UUID and mirrors id: GitHub has no separate installation resource, " +
		"so the Orca GitHub App installation is itself the account-level unit.",
	CollectionKey:  "accounts",
	ListErrorTitle: "Error listing GitHub accounts",
	Attrs: shift_left_common.MergeMaps(shift_left_integration.SharedScmListUnitAttrs(), map[string]dschema.Attribute{
		"id":         dschema.StringAttribute{Computed: true, Description: "Orca UUID of the GitHub account unit."},
		"account_id": dschema.StringAttribute{Computed: true, Description: "Orca UUID of the account (mirrors `id`; GitHub has no separate installation resource)."},
		"github_installation_id": dschema.Int64Attribute{
			Computed:    true,
			Description: "Numeric GitHub App installation id (from GitHub, not minted by Orca).",
		},
		"github_app_settings_url": dschema.StringAttribute{
			Computed:    true,
			Description: "GitHub URL for managing the App installation, when reported.",
		},
	}),
	AttrTypes: shift_left_common.MergeMaps(shift_left_integration.SharedScmListUnitAttrTypes(), map[string]attr.Type{
		"id":                      types.StringType,
		"account_id":              types.StringType,
		"github_installation_id":  types.Int64Type,
		"github_app_settings_url": types.StringType,
	}),
	List: func(c *api_client.APIClient) ([]api_client.GithubInstallation, error) {
		return c.ListGithubInstallations()
	},
	Row: func(a *api_client.GithubInstallation) map[string]attr.Value {
		return shift_left_common.MergeMaps(
			shift_left_integration.SharedScmListUnitValues(a.AccountName, a.ScmUnitCommonFields),
			map[string]attr.Value{
				"id":                      types.StringValue(a.ID),
				"account_id":              types.StringValue(a.ID),
				"github_installation_id":  types.Int64Value(a.GithubInstallationID),
				"github_app_settings_url": tfconv.StringOrNull(a.GithubAppSettingsURL),
			})
	},
}

func NewAccountsDataSource() datasource.DataSource {
	return shift_left_common.NewScmObjectListDataSource(accountsSpec)
}

func accountsToListValue(accs []api_client.GithubInstallation) (types.List, diag.Diagnostics) {
	return accountsSpec.ListValue(accs)
}
