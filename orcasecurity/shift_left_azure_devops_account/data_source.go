package shift_left_azure_devops_account

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

var accountsSpec = shift_left_common.ScmObjectListSpec[api_client.AzureDevopsAccount]{
	TypeNameSuffix: "_shift_left_azure_devops_accounts",
	Description:    "Lists all Orca Azure DevOps shift-left integrated accounts for fleet-wide for_each. Use account_name with orcasecurity_shift_left_azure_devops_account.",
	CollectionKey:  "accounts",
	ListErrorTitle: "Error listing Azure DevOps accounts",
	Attrs: shift_left_common.MergeMaps(shift_left_integration.SharedScmListUnitAttrs(), map[string]dschema.Attribute{
		"id":              dschema.StringAttribute{Computed: true, Description: "Orca UUID of the Azure DevOps account unit."},
		"installation_id": dschema.StringAttribute{Computed: true, Description: "Orca UUID of the parent Azure DevOps installation."},
	}),
	AttrTypes: shift_left_common.MergeMaps(shift_left_integration.SharedScmListUnitAttrTypes(), map[string]attr.Type{
		"id":              types.StringType,
		"installation_id": types.StringType,
	}),
	List: func(c *api_client.APIClient) ([]api_client.AzureDevopsAccount, error) {
		return c.ListAzureDevopsAccounts()
	},
	Row: func(a *api_client.AzureDevopsAccount) map[string]attr.Value {
		return shift_left_common.MergeMaps(
			shift_left_integration.SharedScmListUnitValues(a.AccountName, a.ScmUnitCommonFields),
			map[string]attr.Value{
				"id":              types.StringValue(a.ID),
				"installation_id": types.StringValue(a.InstallationID),
			})
	},
}

func NewAccountsDataSource() datasource.DataSource {
	return shift_left_common.NewScmObjectListDataSource(accountsSpec)
}

func accountsToListValue(accs []api_client.AzureDevopsAccount) (types.List, diag.Diagnostics) {
	return accountsSpec.ListValue(accs)
}
