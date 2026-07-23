package shift_left_bitbucket_account

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

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
	resp.TypeName = req.ProviderTypeName + "_shift_left_bitbucket_accounts"
}

func (ds *accountsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

var accountAttrTypes = map[string]attr.Type{
	"id":                types.StringType,
	"installation_id":   types.StringType,
	"account_id":        types.StringType,
	"account_name":      types.StringType,
	"installation_mode": types.StringType,
	"default_policies":  types.BoolType,
}

func (ds *accountsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "Lists all Orca Bitbucket shift-left integrated accounts across every installation, for fleet-wide for_each.",
		Attributes: map[string]dschema.Attribute{
			"accounts": dschema.ListNestedAttribute{
				Computed: true,
				NestedObject: dschema.NestedAttributeObject{
					Attributes: map[string]dschema.Attribute{
						"id":                dschema.StringAttribute{Computed: true},
						"installation_id":   dschema.StringAttribute{Computed: true},
						"account_id":        dschema.StringAttribute{Computed: true},
						"account_name":      dschema.StringAttribute{Computed: true},
						"installation_mode": dschema.StringAttribute{Computed: true},
						"default_policies":  dschema.BoolAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func accountsToListValue(accs []api_client.BitbucketAccount) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	elemType := types.ObjectType{AttrTypes: accountAttrTypes}
	elems := make([]attr.Value, len(accs))
	for i, a := range accs {
		obj, d := types.ObjectValue(accountAttrTypes, map[string]attr.Value{
			"id":                types.StringValue(a.ID),
			"installation_id":   types.StringValue(a.InstallationID),
			"account_id":        types.StringValue(a.ID),
			"account_name":      types.StringValue(a.AccountName),
			"installation_mode": types.StringValue(a.InstallationMode),
			"default_policies":  types.BoolValue(a.DefaultPolicies),
		})
		diags.Append(d...)
		elems[i] = obj
	}
	list, d := types.ListValue(elemType, elems)
	diags.Append(d...)
	return list, diags
}

func (ds *accountsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	accs, err := ds.apiClient.ListBitbucketAccounts()
	if err != nil {
		resp.Diagnostics.AddError("Error listing Bitbucket accounts", err.Error())
		return
	}
	list, diags := accountsToListValue(accs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &accountsModel{Accounts: list})...)
}
