package system_sonar_alert

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                     = &systemSonarAlertResource{}
	_ resource.ResourceWithConfigure        = &systemSonarAlertResource{}
	_ resource.ResourceWithImportState      = &systemSonarAlertResource{}
	_ resource.ResourceWithConfigValidators = &systemSonarAlertResource{}
)

type systemSonarAlertResource struct {
	apiClient *api_client.APIClient
}

type stateModel struct {
	ID      types.String `tfsdk:"id"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

func NewSystemSonarAlertResource() resource.Resource {
	return &systemSonarAlertResource{}
}

func (r *systemSonarAlertResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_system_sonar_alert"
}

func (r *systemSonarAlertResource) Configure(_ context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *systemSonarAlertResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{}
}

func (r *systemSonarAlertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *systemSonarAlertResource) Schema(_ context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Enables or disable a system sonar-based alert.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Alert rule ID.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Is the alert enabled or disabled.",
				Required:    true,
			},
		},
	}
}

func (r *systemSonarAlertResource) changeSystemSonarAlertStatus(plan *stateModel, diags diag.Diagnostics) {
	changeStatusReq := api_client.SystemAlert{
		ID:      plan.ID.ValueString(),
		Enabled: plan.Enabled.ValueBool(),
	}

	instance, err := r.apiClient.ChangeSystemSonarAlertStatus(changeStatusReq)
	if err != nil {
		diags.AddError(
			"Error changing status for Alert",
			"Could not change status for Alert, unexpected error: "+err.Error(),
		)
		return
	}

	changeResp, err := r.apiClient.GetSystemSonarAlert(instance.ID)
	if err != nil {
		diags.AddError(
			"Error refreshing Alert",
			"Could not change status for Alert, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(changeResp.ID)
	plan.Enabled = types.BoolValue(changeResp.Enabled)
}

func (r *systemSonarAlertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan stateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.changeSystemSonarAlertStatus(&plan, resp.Diagnostics)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *systemSonarAlertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.GetSystemSonarAlert(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Alert",
			fmt.Sprintf("Could not read Alert ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("Alert %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(instance.ID)
	state.Enabled = types.BoolValue(instance.Enabled)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *systemSonarAlertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan stateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.changeSystemSonarAlertStatus(&plan, resp.Diagnostics)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *systemSonarAlertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Upon deletion, we re-enable the system alert because it is unmanaged and enabled is the default state
	state.Enabled = types.BoolValue(true)
	r.changeSystemSonarAlertStatus(&state, resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}
}
