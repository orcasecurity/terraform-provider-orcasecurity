package trusted_dynamic_ip_range

import (
	"context"
	"fmt"
	"strconv"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &trustedDynamicIpRangeResource{}
	_ resource.ResourceWithConfigure   = &trustedDynamicIpRangeResource{}
	_ resource.ResourceWithImportState = &trustedDynamicIpRangeResource{}
)

type trustedDynamicIpRangeResource struct {
	apiClient *api_client.APIClient
}

type trustedDynamicIpRangeResourceModel struct {
	ID      types.String `tfsdk:"id"`
	OrgID   types.String `tfsdk:"org_id"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

func NewTrustedDynamicIpRangeResource() resource.Resource {
	return &trustedDynamicIpRangeResource{}
}

func (r *trustedDynamicIpRangeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dynamic_trusted_ip_range"
}

func (r *trustedDynamicIpRangeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *trustedDynamicIpRangeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing trusted dynamic ip range",
			"Could not convert ID to int64: "+err.Error(),
		)
		return
	}

	// Set all attributes in state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func (r *trustedDynamicIpRangeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a trusted dynamic ip range.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": schema.StringAttribute{
				Required:    true,
				Description: "Orca Identifier for the organization.",
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the dynamic trusted IP range is enabled.",
				Required:    true,
			},
		},
	}
}

// setEnabledState sets or unsets the dynamic trusted IP range based on the enabled value
func (r *trustedDynamicIpRangeResource) setEnabledState(orgID string, enabled bool) error {
	if enabled {
		_, err := r.apiClient.SetTrustedDynamicIpRange(orgID)
		return err
	}
	return r.apiClient.UnsetTrustedDynamicIpRange(orgID)
}

// Create creates the resource and sets the initial Terraform state
func (r *trustedDynamicIpRangeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating resource", "API client not configured. Please configure the provider first.")
		return
	}

	var plan trustedDynamicIpRangeResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.setEnabledState(plan.OrgID.ValueString(), plan.Enabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("Error setting dynamic trusted IP range", fmt.Sprintf("Could not set enabled state: %s", err))
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("toggle_setting_%s", plan.OrgID.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *trustedDynamicIpRangeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state trustedDynamicIpRangeResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get toggle value from backend
	enabled, err := r.apiClient.GetTrustedDynamicIpRangeStatus(state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading toggle setting",
			fmt.Sprintf("Could not read toggle setting: %s", err),
		)
		return
	}

	// Update state
	state.Enabled = types.BoolValue(enabled)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *trustedDynamicIpRangeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan trustedDynamicIpRangeResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.setEnabledState(plan.OrgID.ValueString(), plan.Enabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("Error updating dynamic trusted IP range", fmt.Sprintf("Could not update enabled state: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *trustedDynamicIpRangeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error deleting resource", "API client not configured")
		return
	}

	var state trustedDynamicIpRangeResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.UnsetTrustedDynamicIpRange(state.OrgID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting dynamic trusted IP range", fmt.Sprintf("Could not delete: %v", err))
	}
}
