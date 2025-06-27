package role

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &roleDataSource{}
	_ datasource.DataSourceWithConfigure = &roleDataSource{}
)

type roleStateModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	PermissionGroups types.Set    `tfsdk:"permission_groups"`
	Description      types.String `tfsdk:"description"`
	IsCustom         types.Bool   `tfsdk:"is_custom"`
	CreatedById      types.String `tfsdk:"created_by_id"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

type roleDataSource struct {
	apiClient *api_client.APIClient
}

func NewRoleDataSource() datasource.DataSource {
	return &roleDataSource{}
}

func (ds *roleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *roleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (ds *roleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch role by name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Role ID.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Role name to search for.",
			},
			"permission_groups": schema.SetAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of permission groups assigned to this role.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Role description.",
			},
			"is_custom": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is a custom role or a system-defined role.",
			},
			"created_by_id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the user who created this role.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the role was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the role was last updated.",
			},
		},
	}
}

func (ds *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state roleStateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	item, err := ds.apiClient.GetRoleByName(state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read role", err.Error())
		return
	}

	state.ID = types.StringValue(item.ID)
	state.Name = types.StringValue(item.Name)
	state.Description = types.StringValue(item.Description)
	state.IsCustom = types.BoolValue(item.IsCustom)
	state.CreatedAt = types.StringValue(item.CreatedAt)
	state.UpdatedAt = types.StringValue(item.UpdatedAt)

	// Handle created_by_id (can be null for system roles)
	if item.CreatedBy != nil {
		state.CreatedById = types.StringValue(item.CreatedBy.ID)
	} else {
		state.CreatedById = types.StringNull()
	}

	// Deduplicate permission groups before creating set
	permissionMap := make(map[string]bool)
	var uniquePermissions []string
	for _, perm := range item.PermissionGroups {
		if !permissionMap[perm] {
			permissionMap[perm] = true
			uniquePermissions = append(uniquePermissions, perm)
		}
	}

	// Convert unique permission groups to set
	permissionElements := make([]attr.Value, len(uniquePermissions))
	for i, perm := range uniquePermissions {
		permissionElements[i] = types.StringValue(perm)
	}
	permissionSet, diags := types.SetValue(types.StringType, permissionElements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.PermissionGroups = permissionSet
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
