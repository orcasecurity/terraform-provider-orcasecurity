package custom_role

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &customRoleResource{}
	_ resource.ResourceWithConfigure   = &customRoleResource{}
	_ resource.ResourceWithImportState = &customRoleResource{}
)

type customRoleResource struct {
	apiClient *api_client.APIClient
}

/*type createdByResourceModel struct {
	ID        types.String `tfsdk:"id"`
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
}*/

type customRoleResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	PermissionGroups types.Set    `tfsdk:"permission_groups"`
	//ExpirationDate   types.String `tfsdk:"expiration_date"`
	Description types.String `tfsdk:"description"`
	//CreatedBy        createdByResourceModel `tfsdk:"created_by"`
}

func NewCustomRoleResource() resource.Resource {
	return &customRoleResource{}
}

func (r *customRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_role"
}

func (r *customRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *customRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *customRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	//tflog.Error(ctx, "Setting up Schema")
	resp.Schema = schema.Schema{
		Description: "Provides a custom role resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Custom role ID.",
			},
			"name": schema.StringAttribute{
				Description: "Custom role name. Must be unique across your Orca org.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"permission_groups": schema.SetAttribute{
				Description: "Permissions to assign to the group. Possible permissions",
				ElementType: types.StringType,
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Custom role description.",
				Required:    true,
			},
			/*"expiration_date": schema.StringAttribute{
				Description: "NOT YET SUPPORTED. Date and Time (expressed in Zulu format) after which the role should no longer be active/usable.",
				Optional:    true,
			},
			"created_by": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "ID of user/entity that created this role.",
						Computed:    true,
					},
					"first_name": schema.StringAttribute{
						Description: "First name of user/entity that created this role.",
						Computed:    true,
					},
					"last_name": schema.StringAttribute{
						Description: "Last name of user/entity that created this role.",
						Computed:    true,
					},
				},
			},*/
		},
	}
}

func (r *customRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customRoleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var permissions []string

	for _, item := range plan.PermissionGroups.Elements() {
		permissions = append(permissions, item.String()[1:len(item.String())-1])
	}

	createReq := api_client.CustomRole{
		Name:             plan.Name.ValueString(),
		PermissionGroups: permissions,
		Description:      plan.Description.ValueString(),
	}

	instance, err := r.apiClient.CreateCustomRole(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating custom role",
			"Could not create custom role, unexpected error: "+err.Error(),
		)
		return
	}

	instance, err = r.apiClient.GetCustomRole(instance.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error refreshing custom role",
			"Could not create custom role, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customRoleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.apiClient.DoesCustomRoleExist(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading custom role",
			fmt.Sprintf("Could not read custom role ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	if !exists {
		tflog.Warn(ctx, fmt.Sprintf("Custom role %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	instance, err := r.apiClient.GetCustomRole(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading custom role",
			fmt.Sprintf("Could not read custom role ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	state.ID = types.StringValue(instance.ID)
	state.Description = types.StringValue(instance.Description)
	state.Name = types.StringValue(instance.Name)
	/*state.CreatedBy.ID = types.StringValue(instance.CreatedBy.ID)
	state.CreatedBy.FirstName = types.StringValue(instance.CreatedBy.FirstName)
	state.CreatedBy.LastName = types.StringValue(instance.CreatedBy.LastName)*/

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customRoleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var permissions []string

	for _, item := range plan.PermissionGroups.Elements() {
		permissions = append(permissions, item.String()[1:len(item.String())-1])
	}

	updateReq := api_client.CustomRole{
		ID:               plan.ID.ValueString(),
		Name:             plan.Name.ValueString(),
		PermissionGroups: permissions,
		Description:      plan.Description.ValueString(),
	}

	_, err := r.apiClient.UpdateCustomRole(updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating custom role",
			"Could not update custom role, unexpected error: "+err.Error(),
		)
		return
	}

	instance, err := r.apiClient.GetCustomRole(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating custom role",
			"Could not read custom role, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.Description = types.StringValue(instance.Description)
	plan.Name = types.StringValue(instance.Name)
	/*plan.CreatedBy.ID = types.StringValue(instance.CreatedBy.ID)
	plan.CreatedBy.FirstName = types.StringValue(instance.CreatedBy.FirstName)
	plan.CreatedBy.LastName = types.StringValue(instance.CreatedBy.LastName)*/

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customRoleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteCustomRole(state.ID.String()[1 : len(state.ID.String())-1])
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting custom role",
			"Could not delete custom role, unexpected error: "+err.Error(),
		)
		return
	}
}
