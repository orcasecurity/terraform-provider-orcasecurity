package shift_left_azure_devops_account

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var accountsSpec = shift_left_integration.ScmUnitListSpec[api_client.AzureDevopsAccount]{
	TypeNameSuffix: "_shift_left_azure_devops_accounts",
	Description:    "Lists all Orca Azure DevOps shift-left integrated accounts for fleet-wide for_each.",
	CollectionKey:  "accounts",
	ListErrorTitle: "Error listing Azure DevOps accounts",
	Extra: map[string]attr.Type{
		"id":              types.StringType,
		"installation_id": types.StringType,
		"account_id":      types.StringType,
	},
	List: func(c *api_client.APIClient) ([]api_client.AzureDevopsAccount, error) {
		return c.ListAzureDevopsAccounts()
	},
	Row: func(a *api_client.AzureDevopsAccount) (string, api_client.ScmUnitCommonFields, map[string]attr.Value) {
		return a.AccountName, a.ScmUnitCommonFields, map[string]attr.Value{
			"id":              types.StringValue(a.ID),
			"installation_id": types.StringValue(a.InstallationID),
			"account_id":      types.StringValue(a.ID),
		}
	},
}

func NewAccountsDataSource() datasource.DataSource {
	return shift_left_integration.NewScmUnitListDataSource(accountsSpec)
}

func accountsToListValue(accs []api_client.AzureDevopsAccount) (types.List, diag.Diagnostics) {
	return accountsSpec.ListValue(accs)
}
