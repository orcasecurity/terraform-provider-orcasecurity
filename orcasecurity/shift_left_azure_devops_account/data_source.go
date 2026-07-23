package shift_left_azure_devops_account

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &accountsDataSource{}
	_ datasource.DataSourceWithConfigure = &accountsDataSource{}
)

type accountsDataSource struct {
	apiClient *api_client.APIClient
}

type accountsModel struct {
	Accounts types.List `tfsdk:"accounts"`
}

func NewAccountsDataSource() datasource.DataSource { return &accountsDataSource{} }

func (ds *accountsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_azure_devops_accounts"
}

func (ds *accountsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func accountAttrTypes() map[string]attr.Type {
	attrs := shift_left_integration.SharedScmListUnitAttrTypes()
	attrs["id"] = types.StringType
	attrs["installation_id"] = types.StringType
	attrs["account_id"] = types.StringType
	return attrs
}

func (ds *accountsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	nested := shift_left_integration.SharedScmListUnitAttrs()
	nested["id"] = dschema.StringAttribute{Computed: true}
	nested["installation_id"] = dschema.StringAttribute{Computed: true}
	nested["account_id"] = dschema.StringAttribute{Computed: true}
	resp.Schema = dschema.Schema{
		Description: "Lists all Orca Azure DevOps shift-left integrated accounts for fleet-wide for_each.",
		Attributes: map[string]dschema.Attribute{
			"accounts": dschema.ListNestedAttribute{
				Computed: true,
				NestedObject: dschema.NestedAttributeObject{
					Attributes: nested,
				},
			},
		},
	}
}

func accountsToListValue(accs []api_client.AzureDevopsAccount) (types.List, diag.Diagnostics) {
	attrTypes := accountAttrTypes()
	elems := make([]map[string]attr.Value, len(accs))
	for i, a := range accs {
		m := shift_left_integration.SharedScmListUnitValues(a.AccountName, a.ScmUnitCommonFields)
		m["id"] = types.StringValue(a.ID)
		m["installation_id"] = types.StringValue(a.InstallationID)
		m["account_id"] = types.StringValue(a.ID)
		elems[i] = m
	}
	return shift_left_integration.ObjectListFromValues(attrTypes, elems)
}

func (ds *accountsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	accs, err := ds.apiClient.ListAzureDevopsAccounts()
	if err != nil {
		resp.Diagnostics.AddError("Error listing Azure DevOps accounts", err.Error())
		return
	}
	list, diags := accountsToListValue(accs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &accountsModel{Accounts: list})...)
}
