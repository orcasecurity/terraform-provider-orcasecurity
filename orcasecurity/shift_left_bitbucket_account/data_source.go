package shift_left_bitbucket_account

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var accountsSpec = shift_left_integration.ScmUnitListSpec[api_client.BitbucketAccount]{
	TypeNameSuffix: "_shift_left_bitbucket_accounts",
	Description:    "Lists all Orca Bitbucket shift-left integrated accounts for fleet-wide for_each. account_id is the Bitbucket slug.",
	CollectionKey:  "accounts",
	ListErrorTitle: "Error listing Bitbucket accounts",
	Extra: map[string]attr.Type{
		"id":              types.StringType,
		"installation_id": types.StringType,
		"account_id":      types.StringType,
	},
	List: func(c *api_client.APIClient) ([]api_client.BitbucketAccount, error) {
		return c.ListBitbucketAccounts()
	},
	Row: func(a *api_client.BitbucketAccount) (string, api_client.ScmUnitCommonFields, map[string]attr.Value) {
		return a.AccountName, a.ScmUnitCommonFields, map[string]attr.Value{
			"id":              types.StringValue(a.ID),
			"installation_id": types.StringValue(a.InstallationID),
			"account_id":      types.StringValue(a.AccountID),
		}
	},
}

func NewAccountsDataSource() datasource.DataSource {
	return shift_left_integration.NewScmUnitListDataSource(accountsSpec)
}

func accountsToListValue(accs []api_client.BitbucketAccount) (types.List, diag.Diagnostics) {
	return accountsSpec.ListValue(accs)
}
