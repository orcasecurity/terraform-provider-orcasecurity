package rbac_role

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &rbacRolesDataSource{}
	_ datasource.DataSourceWithConfigure = &rbacRolesDataSource{}
)

type rbacRolesDataSource struct {
	apiClient *api_client.APIClient
}

func NewRbacRolesDataSource() datasource.DataSource {
	return &rbacRolesDataSource{}
}

func (ds *rbacRolesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *rbacRolesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rbac_roles"
}

func (ds *rbacRolesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all RBAC roles from GET /api/rbac/role. Pick a role id with a for expression or one([for r in ... : r.id if r.name == \"Viewer\"]).",
		Attributes: map[string]schema.Attribute{
			"roles": schema.ListNestedAttribute{
				Description: "Each object is one role from the API.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Role id (use with orcasecurity_group_access.role_id).",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Display name of the role.",
						},
					},
				},
			},
		},
	}
}

func rolesToListValue(roles []api_client.RBACRole) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	attrTypes := map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
	}
	elemType := types.ObjectType{AttrTypes: attrTypes}

	elems := make([]attr.Value, len(roles))
	for i, r := range roles {
		obj, d := types.ObjectValue(attrTypes, map[string]attr.Value{
			"id":   types.StringValue(r.ID),
			"name": types.StringValue(r.Name),
		})
		diags.Append(d...)
		elems[i] = obj
	}
	if diags.HasError() {
		return types.ListNull(elemType), diags
	}
	listVal, d := types.ListValue(elemType, elems)
	diags.Append(d...)
	return listVal, diags
}

type rbacRolesDataSourceModel struct {
	Roles types.List `tfsdk:"roles"`
}

func (ds *rbacRolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	roles, err := ds.apiClient.ListRBACRoles()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read RBAC roles", err.Error())
		return
	}

	listVal, diags := rolesToListValue(roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := rbacRolesDataSourceModel{Roles: listVal}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
