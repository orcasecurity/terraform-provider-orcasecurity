package group

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &groupDataSource{}
	_ datasource.DataSourceWithConfigure = &groupDataSource{}
)

type groupStateModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	SSOGroup    types.Bool   `tfsdk:"sso_group"`
	Users       types.Set    `tfsdk:"users"`
}

type groupDataSource struct {
	apiClient *api_client.APIClient
}

func NewGroupDataSource() datasource.DataSource {
	return &groupDataSource{}
}

func (ds *groupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *groupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (ds *groupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch group by name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Group ID.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Group name to search for.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Group description.",
			},
			"sso_group": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this group may be used for SSO permissions.",
			},
			"users": schema.SetAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of user IDs in this group.",
			},
		},
	}
}

func (ds *groupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state groupStateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	item, err := ds.apiClient.GetGroupByName(state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read group", err.Error())
		return
	}

	state.ID = types.StringValue(item.ID)
	state.Name = types.StringValue(item.Name)
	state.Description = types.StringValue(item.Description)
	state.SSOGroup = types.BoolValue(item.SSOGroup)

	// Convert users slice to set
	userElements := make([]attr.Value, len(item.Users))
	for i, user := range item.Users {
		userElements[i] = types.StringValue(user)
	}
	userSet, diags := types.SetValue(types.StringType, userElements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Users = userSet
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
