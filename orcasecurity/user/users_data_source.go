package user

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
	_ datasource.DataSource              = &usersDataSource{}
	_ datasource.DataSourceWithConfigure = &usersDataSource{}
)

type usersDataSource struct {
	apiClient *api_client.APIClient
}

func NewUsersDataSource() datasource.DataSource {
	return &usersDataSource{}
}

func (ds *usersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *usersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (ds *usersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all users in the organization from GET /api/users. Pick a user_id with a for expression, e.g. one([for u in data.orcasecurity_users.all.users : u.user_id if u.email == \"jane@example.com\"]).",
		Attributes: map[string]schema.Attribute{
			"users": schema.ListNestedAttribute{
				Description: "Each object is one user from the API.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user_id": schema.StringAttribute{
							Computed:    true,
							Description: "User id (use with orcasecurity_user_access.user_id and orcasecurity_group.users).",
						},
						"email": schema.StringAttribute{
							Computed:    true,
							Description: "User email address.",
						},
						"first_name": schema.StringAttribute{
							Computed:    true,
							Description: "First name.",
						},
						"last_name": schema.StringAttribute{
							Computed:    true,
							Description: "Last name.",
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: "Account status (e.g. active, invited).",
						},
						"mfa_required": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether MFA is required for this user.",
						},
						"mfa_enabled": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the user has an MFA device enrolled.",
						},
					},
				},
			},
		},
	}
}

var userAttrTypes = map[string]attr.Type{
	"user_id":      types.StringType,
	"email":        types.StringType,
	"first_name":   types.StringType,
	"last_name":    types.StringType,
	"status":       types.StringType,
	"mfa_required": types.BoolType,
	"mfa_enabled":  types.BoolType,
}

func usersToListValue(users []api_client.User) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	elemType := types.ObjectType{AttrTypes: userAttrTypes}

	elems := make([]attr.Value, len(users))
	for i, u := range users {
		obj, d := types.ObjectValue(userAttrTypes, map[string]attr.Value{
			"user_id":      types.StringValue(u.ID),
			"email":        types.StringValue(u.Email),
			"first_name":   types.StringValue(u.FirstName),
			"last_name":    types.StringValue(u.LastName),
			"status":       types.StringValue(u.Status),
			"mfa_required": types.BoolValue(u.MFARequired),
			"mfa_enabled":  types.BoolValue(u.MFAEnabled),
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

type usersDataSourceModel struct {
	Users types.List `tfsdk:"users"`
}

func (ds *usersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	users, err := ds.apiClient.ListUsers()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read users", err.Error())
		return
	}

	listVal, diags := usersToListValue(users)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := usersDataSourceModel{Users: listVal}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
