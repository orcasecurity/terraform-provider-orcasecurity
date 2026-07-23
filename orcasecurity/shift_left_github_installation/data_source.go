package shift_left_github_installation

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
	_ datasource.DataSource              = &installationsDataSource{}
	_ datasource.DataSourceWithConfigure = &installationsDataSource{}
)

type installationsDataSource struct {
	apiClient *api_client.APIClient
}

type installationsModel struct {
	Installations types.List `tfsdk:"installations"`
}

func NewInstallationsDataSource() datasource.DataSource { return &installationsDataSource{} }

func (ds *installationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_github_installations"
}

func (ds *installationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

var installationAttrTypes = map[string]attr.Type{
	"id":                types.StringType,
	"account_name":      types.StringType,
	"installation_mode": types.StringType,
	"default_policies":  types.BoolType,
}

func (ds *installationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "Lists all Orca GitHub shift-left installations for fleet-wide for_each.",
		Attributes: map[string]dschema.Attribute{
			"installations": dschema.ListNestedAttribute{
				Computed: true,
				NestedObject: dschema.NestedAttributeObject{
					Attributes: map[string]dschema.Attribute{
						"id":                dschema.StringAttribute{Computed: true},
						"account_name":      dschema.StringAttribute{Computed: true},
						"installation_mode": dschema.StringAttribute{Computed: true},
						"default_policies":  dschema.BoolAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func installationsToListValue(insts []api_client.GithubInstallation) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	elemType := types.ObjectType{AttrTypes: installationAttrTypes}
	elems := make([]attr.Value, len(insts))
	for i, in := range insts {
		obj, d := types.ObjectValue(installationAttrTypes, map[string]attr.Value{
			"id":                types.StringValue(in.ID),
			"account_name":      types.StringValue(in.AccountName),
			"installation_mode": types.StringValue(in.InstallationMode),
			"default_policies":  types.BoolValue(in.DefaultPolicies),
		})
		diags.Append(d...)
		elems[i] = obj
	}
	list, d := types.ListValue(elemType, elems)
	diags.Append(d...)
	return list, diags
}

func (ds *installationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	insts, err := ds.apiClient.ListGithubInstallations()
	if err != nil {
		resp.Diagnostics.AddError("Error listing GitHub installations", err.Error())
		return
	}
	list, diags := installationsToListValue(insts)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &installationsModel{Installations: list})...)
}
