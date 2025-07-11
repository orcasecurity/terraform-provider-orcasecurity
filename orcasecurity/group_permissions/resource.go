package group_permissions

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
	_ resource.Resource                = &groupPermissionResource{}
	_ resource.ResourceWithConfigure   = &groupPermissionResource{}
	_ resource.ResourceWithImportState = &groupPermissionResource{}
)

type groupPermissionResource struct {
	apiClient *api_client.APIClient
}

type groupPermissionResourceModel struct {
	ID               types.String `tfsdk:"id"`
	GroupID          types.String `tfsdk:"group_id"`
	AllCloudAccounts types.Bool   `tfsdk:"all_cloud_accounts"`
	CloudAccounts    types.Set    `tfsdk:"cloud_accounts"`
	RoleID           types.String `tfsdk:"role_id"`
	BusinessUnits    types.Set    `tfsdk:"business_units"`
	ShiftleftProjects types.Set `tfsdk:"shiftleft_projects"`
}

func NewGroupPermissionResource() resource.Resource {
	return &groupPermissionResource{}
}

func (r *groupPermissionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_permission"
}

func (r *groupPermissionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *groupPermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *groupPermissionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a group permission resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Group Permission ID.",
			},
			"group_id": schema.StringAttribute{
				Description: "Group ID. Must be a valid group ID.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"all_cloud_accounts": schema.BoolAttribute{
				Description: "Whether the group has access to all cloud accounts.",
				Required:    true,
			},
			"cloud_accounts": schema.SetAttribute{
				Description: "Cloud accounts the group has access to. Required if all_cloud_accounts is false.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"role_id": schema.StringAttribute{
				Description: "Role ID. Must be a valid role ID.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"business_units": schema.SetAttribute{
				Description: "Business units.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"shiftleft_projects": schema.SetAttribute{
				Description: "Shiftleft projects.",
				ElementType: types.StringType,
				Optional:    true,
			},
		},
	}
}

func (r *groupPermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupPermissionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var cloudAccounts []string
	if !plan.AllCloudAccounts.ValueBool() {
		for _, item := range plan.CloudAccounts.Elements() {
			cloudAccounts = append(cloudAccounts, item.String()[1:len(item.String())-1])
		}
	}

	var businessUnits []string
	for _, item := range plan.BusinessUnits.Elements() {
		businessUnits = append(businessUnits, item.String()[1:len(item.String())-1])
	}

	var shiftleftProjects []string
	for _, item := range plan.ShiftleftProjects.Elements() {
		shiftleftProjects = append(shiftleftProjects, item.String()[1:len(item.String())-1])
	}

	createReq := api_client.GroupPermission{
		Group: api_client.GroupInfo{
			ID: plan.GroupID.ValueString(),
		},
		AllCloudAccounts: plan.AllCloudAccounts.ValueBool(),
		CloudAccounts:    cloudAccounts,
		Role: api_client.RoleInfo{
			ID: plan.RoleID.ValueString(),
		},
		UserFilters:       businessUnits,
		ShiftleftProjects: shiftleftProjects,
	}

	instance, err := r.apiClient.CreateGroupPermission(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating group permission",
			"Could not create group permission, unexpected error: "+err.Error(),
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

func (r *groupPermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state groupPermissionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.GetGroupPermission(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading group permission",
			fmt.Sprintf("Could not read group permission ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("Group permission %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(instance.ID)
	state.GroupID = types.StringValue(instance.Group.ID)
	state.AllCloudAccounts = types.BoolValue(instance.AllCloudAccounts)
	state.RoleID = types.StringValue(instance.Role.ID)

	// Sets can be tricky, we'll just overwrite them
	cloudAccounts, diags := types.SetValueFrom(ctx, types.StringType, instance.CloudAccounts)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.CloudAccounts = cloudAccounts

	businessUnits, diags := types.SetValueFrom(ctx, types.StringType, instance.UserFilters)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.BusinessUnits = businessUnits

	shiftleftProjects, diags := types.SetValueFrom(ctx, types.StringType, instance.ShiftleftProjects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.ShiftleftProjects = shiftleftProjects


	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *groupPermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Error updating group permission",
		"Update operation not supported by API",
	)
}

func (r *groupPermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupPermissionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteGroupPermissionWithBody(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting group permission",
			"Could not delete group permission, unexpected error: "+err.Error(),
		)
		return
	}
}
