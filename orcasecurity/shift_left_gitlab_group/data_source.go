package shift_left_gitlab_group

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
	_ datasource.DataSource              = &groupsDataSource{}
	_ datasource.DataSourceWithConfigure = &groupsDataSource{}
)

type groupsDataSource struct {
	apiClient *api_client.APIClient
}

type groupsModel struct {
	Groups types.List `tfsdk:"groups"`
}

func NewGroupsDataSource() datasource.DataSource { return &groupsDataSource{} }

func (ds *groupsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_gitlab_groups"
}

func (ds *groupsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func groupAttrTypes() map[string]attr.Type {
	attrs := shift_left_integration.SharedScmListUnitAttrTypes()
	attrs["id"] = types.StringType
	attrs["installation_id"] = types.StringType
	attrs["group_id"] = types.StringType
	return attrs
}

func (ds *groupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	nested := shift_left_integration.SharedScmListUnitAttrs()
	nested["id"] = dschema.StringAttribute{Computed: true}
	nested["installation_id"] = dschema.StringAttribute{Computed: true}
	nested["group_id"] = dschema.StringAttribute{Computed: true}
	resp.Schema = dschema.Schema{
		Description: "Lists all Orca GitLab shift-left integrated groups for fleet-wide for_each.",
		Attributes: map[string]dschema.Attribute{
			"groups": dschema.ListNestedAttribute{
				Computed: true,
				NestedObject: dschema.NestedAttributeObject{
					Attributes: nested,
				},
			},
		},
	}
}

func groupsToListValue(grps []api_client.GitlabGroup) (types.List, diag.Diagnostics) {
	attrTypes := groupAttrTypes()
	elems := make([]map[string]attr.Value, len(grps))
	for i, g := range grps {
		m := shift_left_integration.SharedScmListUnitValues(g.AccountName, g.InstallationMode, g.IntegrationStatus, g.DefaultPolicies)
		m["id"] = types.StringValue(g.ID)
		m["installation_id"] = types.StringValue(g.InstallationID)
		m["group_id"] = types.StringValue(g.ID)
		elems[i] = m
	}
	return shift_left_integration.ObjectListFromValues(attrTypes, elems)
}

func (ds *groupsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	grps, err := ds.apiClient.ListGitlabGroups()
	if err != nil {
		resp.Diagnostics.AddError("Error listing GitLab groups", err.Error())
		return
	}
	list, diags := groupsToListValue(grps)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &groupsModel{Groups: list})...)
}
