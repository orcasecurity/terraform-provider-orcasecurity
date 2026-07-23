package shift_left_gitlab_group

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

func (ds *groupsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

var groupAttrTypes = map[string]attr.Type{
	"id":                types.StringType,
	"installation_id":   types.StringType,
	"group_id":          types.StringType,
	"account_name":      types.StringType,
	"installation_mode": types.StringType,
	"default_policies":  types.BoolType,
}

func (ds *groupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "Lists all Orca GitLab shift-left integrated groups for fleet-wide for_each.",
		Attributes: map[string]dschema.Attribute{
			"groups": dschema.ListNestedAttribute{
				Computed: true,
				NestedObject: dschema.NestedAttributeObject{
					Attributes: map[string]dschema.Attribute{
						"id":                dschema.StringAttribute{Computed: true},
						"installation_id":   dschema.StringAttribute{Computed: true},
						"group_id":          dschema.StringAttribute{Computed: true},
						"account_name":      dschema.StringAttribute{Computed: true},
						"installation_mode": dschema.StringAttribute{Computed: true},
						"default_policies":  dschema.BoolAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func groupsToListValue(grps []api_client.GitlabGroup) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	elemType := types.ObjectType{AttrTypes: groupAttrTypes}
	elems := make([]attr.Value, len(grps))
	for i, g := range grps {
		obj, d := types.ObjectValue(groupAttrTypes, map[string]attr.Value{
			"id":                types.StringValue(g.ID),
			"installation_id":   types.StringValue(g.InstallationID),
			"group_id":          types.StringValue(g.ID),
			"account_name":      types.StringValue(g.AccountName),
			"installation_mode": types.StringValue(g.InstallationMode),
			"default_policies":  types.BoolValue(g.DefaultPolicies),
		})
		diags.Append(d...)
		elems[i] = obj
	}
	list, d := types.ListValue(elemType, elems)
	diags.Append(d...)
	return list, diags
}

func (ds *groupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
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
