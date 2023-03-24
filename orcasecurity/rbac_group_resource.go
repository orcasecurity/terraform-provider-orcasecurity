package orcasecurity

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &rbacGroupResource{}
	_ resource.ResourceWithConfigure = &rbacGroupResource{}
)

type rbacGroupResource struct {
	apiClient *api_client.APIClient
}

type rbacGroupResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	SSOGroup    types.Bool   `tfsdk:"sso_group"`
}

// Configure implements resource.ResourceWithConfigure
func (r *rbacGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

// Schema implements resource.Resource
func (r *rbacGroupResource) Schema(_ context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name":        schema.StringAttribute{Required: true},
			"description": schema.StringAttribute{Optional: true},
			"sso_group":   schema.BoolAttribute{Optional: true},
		},
	}
}

// Metadata implements resource.Resource
func (r *rbacGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_rbac_group"
}

// Create implements resource.Resource
func (r *rbacGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan rbacGroupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.apiClient.CreateRBACGroup(api_client.RBACGroup{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		SSOGroup:    plan.SSOGroup.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating RBAC group",
			"Could not create RBAC group, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(group.ID)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete implements resource.Resource
func (r *rbacGroupResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
	panic("unimplemented")
}

// Read implements resource.Resource
func (r *rbacGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state rbacGroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.apiClient.GetRBACGroup(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading RBAC group", "Could not read RBAC Group ID "+state.Name.ValueString()+": "+err.Error())
		return
	}
	state.ID = types.StringValue(group.ID)
	state.Name = types.StringValue(group.Name)
	state.Description = types.StringValue(group.Description)
	state.SSOGroup = types.BoolValue(group.SSOGroup)

	diags = req.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update implements resource.Resource
func (r *rbacGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan rbacGroupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Plan ID is null",
			"Could not update RBAC group, unexpected error: "+plan.ID.ValueString(),
		)
		return
	}

	_, err := r.apiClient.UpdateRBACGroup(
		plan.ID.ValueString(),
		api_client.RBACGroup{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			SSOGroup:    plan.SSOGroup.ValueBool(),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating RBAC group",
			"Could not update RBAC group, unexpected error: "+err.Error(),
		)
		return
	}

	group, err := r.apiClient.GetRBACGroup(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading RBAC group",
			"Could not read RBAC group ID: "+plan.ID.ValueString()+": "+err.Error(),
		)
	}
	plan.Name = types.StringValue(group.Name)
	plan.Description = types.StringValue(group.Description)
	plan.SSOGroup = types.BoolValue(group.SSOGroup)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

func NewRBACGroupResource() resource.Resource {
	return &rbacGroupResource{}
}
