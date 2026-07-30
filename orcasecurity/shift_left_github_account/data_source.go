package shift_left_github_account

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var accountsSpec = shift_left_integration.ScmUnitListSpec[api_client.GithubInstallation]{
	TypeNameSuffix: "_shift_left_github_accounts",
	Description: "Lists all Orca GitHub shift-left integrated accounts for fleet-wide for_each. " +
		"account_id is the Orca UUID and mirrors id: GitHub has no separate installation resource, " +
		"so the Orca GitHub App installation is itself the account-level unit.",
	CollectionKey:  "accounts",
	ListErrorTitle: "Error listing GitHub accounts",
	Extra: map[string]attr.Type{
		"id":                      types.StringType,
		"account_id":              types.StringType,
		"github_installation_id":  types.Int64Type,
		"github_app_settings_url": types.StringType,
	},
	List: func(c *api_client.APIClient) ([]api_client.GithubInstallation, error) {
		return c.ListGithubInstallations()
	},
	Row: func(a *api_client.GithubInstallation) (string, api_client.ScmUnitCommonFields, map[string]attr.Value) {
		return a.AccountName, a.ScmUnitCommonFields, map[string]attr.Value{
			"id":                      types.StringValue(a.ID),
			"account_id":              types.StringValue(a.ID),
			"github_installation_id":  types.Int64Value(a.GithubInstallationID),
			"github_app_settings_url": tfconv.StringOrNull(a.GithubAppSettingsURL),
		}
	},
}

func NewAccountsDataSource() datasource.DataSource {
	return shift_left_integration.NewScmUnitListDataSource(accountsSpec)
}

func accountsToListValue(accs []api_client.GithubInstallation) (types.List, diag.Diagnostics) {
	return accountsSpec.ListValue(accs)
}
