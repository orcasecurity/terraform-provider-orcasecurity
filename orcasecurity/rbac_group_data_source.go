package orcasecurity

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &rbacGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &rbacGroupDataSource{}
)

type rbacGroupDataSourceModel struct {
	RBACGroups []rbacGroupModel `tfsdk:"rbac_groups"`
}

type rbacGroupModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	SSOGroup    types.Bool   `tfsdk:"sso_group"`
}

type rbacGroupDataSource struct {
	apiClient *api_client.APIClient
}

func (ds *rbacGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *rbacGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rbac_group"
}

func (ds *rbacGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state rbacGroupDataSourceModel

	rbacGroups, err := ds.apiClient.GetRBACGroups()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Orca Security RBAC groups", err.Error())
		return
	}

	for _, group := range rbacGroups {
		groupState := rbacGroupModel{
			ID:          types.StringValue(group.ID),
			Name:        types.StringValue(group.Name),
			Description: types.StringValue(group.Description),
			SSOGroup:    types.BoolValue(group.SSOGroup),
		}
		state.RBACGroups = append(state.RBACGroups, groupState)
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (ds *rbacGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch RBAC group list.",
		Attributes: map[string]schema.Attribute{
			"rbac_groups": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Object ID",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Group name",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Group description",
							Computed:    true,
						},
						"sso_group": schema.BoolAttribute{
							Description: "Whether the group should be used only with SSO.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func NewRBACGroupDataSource() datasource.DataSource {
	return &rbacGroupDataSource{}
}
