package shift_left_github_installation

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

func (ds *installationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func installationAttrTypes() map[string]attr.Type {
	attrs := shift_left_integration.SharedScmListUnitAttrTypes()
	attrs["id"] = types.StringType
	attrs["installation_id"] = types.StringType
	attrs["github_installation_id"] = types.Int64Type
	attrs["github_app_settings_url"] = types.StringType
	return attrs
}

func (ds *installationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	nested := shift_left_integration.SharedScmListUnitAttrs()
	nested["id"] = dschema.StringAttribute{Computed: true}
	nested["installation_id"] = dschema.StringAttribute{Computed: true}
	nested["github_installation_id"] = dschema.Int64Attribute{Computed: true}
	nested["github_app_settings_url"] = dschema.StringAttribute{Computed: true}
	resp.Schema = dschema.Schema{
		Description: "Lists all Orca GitHub shift-left installations for fleet-wide for_each.",
		Attributes: map[string]dschema.Attribute{
			"installations": dschema.ListNestedAttribute{
				Computed: true,
				NestedObject: dschema.NestedAttributeObject{
					Attributes: nested,
				},
			},
		},
	}
}

func installationsToListValue(insts []api_client.GithubInstallation) (types.List, diag.Diagnostics) {
	attrTypes := installationAttrTypes()
	elems := make([]map[string]attr.Value, len(insts))
	for i, in := range insts {
		m := shift_left_integration.SharedScmListUnitValues(in.AccountName, in.ScmUnitCommonFields)
		m["id"] = types.StringValue(in.ID)
		m["installation_id"] = types.StringValue(in.ID)
		m["github_installation_id"] = types.Int64Value(in.GithubInstallationID)
		m["github_app_settings_url"] = shift_left_integration.OptionalID(in.GithubAppSettingsURL)
		elems[i] = m
	}
	return shift_left_integration.ObjectListFromValues(attrTypes, elems)
}

func (ds *installationsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
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
