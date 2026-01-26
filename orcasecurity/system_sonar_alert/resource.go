package system_sonar_alert

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &systemSonarAlertResource{}
	_ resource.ResourceWithConfigure   = &systemSonarAlertResource{}
	_ resource.ResourceWithImportState = &systemSonarAlertResource{}
)

type systemSonarAlertResource struct {
	apiClient *api_client.APIClient
}

type stateModel struct {
	RuleID   types.String  `tfsdk:"rule_id"`
	Enabled  types.Bool    `tfsdk:"enabled"`
	Name     types.String  `tfsdk:"name"`
	Category types.String  `tfsdk:"category"`
	Score    types.Float64 `tfsdk:"score"`
	RuleType types.String  `tfsdk:"rule_type"`
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

func (r *systemSonarAlertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("rule_id"), req, resp)
}

func (r *systemSonarAlertResource) Schema(_ context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Manages the enabled/disabled state of a built-in Orca system sonar alert. System alerts are pre-defined by Orca and cannot be created or deleted, only enabled or disabled.",
		Attributes: map[string]schema.Attribute{
			"rule_id": schema.StringAttribute{
				Description: "The unique identifier of the system alert rule.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the system alert is enabled.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the system alert.",
				Computed:    true,
			},
			"category": schema.StringAttribute{
				Description: "The category of the system alert.",
				Computed:    true,
			},
			"score": schema.Float64Attribute{
				Description: "The score of the system alert.",
				Computed:    true,
			},
			"rule_type": schema.StringAttribute{
				Description: "The rule type identifier of the system alert.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *systemSonarAlertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan stateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// First, get the alert to ensure it exists and get its details
	alert, err := r.apiClient.GetSystemSonarAlert(plan.RuleID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading system alert",
			fmt.Sprintf("Could not read system alert ID %s: %s", plan.RuleID.ValueString(), err.Error()),
		)
		return
	}

	// Check if this is actually a system alert (not custom)
	if alert.Custom {
		resp.Diagnostics.AddError(
			"Invalid alert type",
			fmt.Sprintf("Alert ID %s is a custom alert, not a system alert. Use orcasecurity_custom_sonar_alert resource instead.", plan.RuleID.ValueString()),
		)
		return
	}

	// Update the alert status
	_, err = r.apiClient.UpdateSystemSonarAlertStatus(
		plan.RuleID.ValueString(),
		alert.RuleType,
		plan.Enabled.ValueBool(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating system alert status",
			fmt.Sprintf("Could not update system alert ID %s: %s", plan.RuleID.ValueString(), err.Error()),
		)
		return
	}

	// Set state with the alert details
	plan.Name = types.StringValue(alert.Name)
	plan.Category = types.StringValue(alert.Category)
	plan.Score = types.Float64Value(alert.Score)
	plan.RuleType = types.StringValue(alert.RuleType)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *systemSonarAlertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.apiClient.DoesSystemSonarAlertExist(state.RuleID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error checking system alert existence",
			fmt.Sprintf("Could not check system alert ID %s: %s", state.RuleID.ValueString(), err.Error()),
		)
		return
	}

	if !exists {
		tflog.Warn(ctx, fmt.Sprintf("System alert %s is missing on the remote side.", state.RuleID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	alert, err := r.apiClient.GetSystemSonarAlert(state.RuleID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading system alert",
			fmt.Sprintf("Could not read system alert ID %s: %s", state.RuleID.ValueString(), err.Error()),
		)
		return
	}

	state.Name = types.StringValue(alert.Name)
	state.Category = types.StringValue(alert.Category)
	state.Score = types.Float64Value(alert.Score)
	state.RuleType = types.StringValue(alert.RuleType)
	state.Enabled = types.BoolValue(alert.Enabled)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *systemSonarAlertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan stateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state stateModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the rule_type from state (it's computed and won't be in plan)
	ruleType := state.RuleType.ValueString()

	// Update the alert status
	_, err := r.apiClient.UpdateSystemSonarAlertStatus(
		plan.RuleID.ValueString(),
		ruleType,
		plan.Enabled.ValueBool(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating system alert status",
			fmt.Sprintf("Could not update system alert ID %s: %s", plan.RuleID.ValueString(), err.Error()),
		)
		return
	}

	// Preserve computed values from state
	plan.Name = state.Name
	plan.Category = state.Category
	plan.Score = state.Score
	plan.RuleType = state.RuleType

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *systemSonarAlertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For system alerts, "delete" means re-enabling the alert (restoring default state)
	// System alerts cannot actually be deleted, they are built-in
	_, err := r.apiClient.UpdateSystemSonarAlertStatus(
		state.RuleID.ValueString(),
		state.RuleType.ValueString(),
		true, // Re-enable on delete
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error restoring system alert status",
			fmt.Sprintf("Could not restore system alert ID %s to enabled state: %s", state.RuleID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Info(ctx, fmt.Sprintf("System alert %s has been re-enabled (restored to default state)", state.RuleID.ValueString()))
}
