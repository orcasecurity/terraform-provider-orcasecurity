package azure_sentinel

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &azureSentinelResource{}
	_ resource.ResourceWithConfigure   = &azureSentinelResource{}
	_ resource.ResourceWithImportState = &azureSentinelResource{}
)

type azureSentinelResource struct {
	apiClient *api_client.APIClient
}

type azureSentinelResourceModel struct {
	ID            types.String `tfsdk:"id"`
	TemplateName  types.String `tfsdk:"template_name"`
	LogType       types.String `tfsdk:"log_type"`
	PrimaryKey    types.String `tfsdk:"primary_key"`
	WorkspaceID   types.String `tfsdk:"workspace_id"`
	IsEnabled     types.Bool   `tfsdk:"is_enabled"`
	IsDefault     types.Bool   `tfsdk:"is_default"`
	BusinessUnits types.Set    `tfsdk:"business_units"`
}

func NewAzureSentinelResource() resource.Resource {
	return &azureSentinelResource{}
}

func (r *azureSentinelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_azure_sentinel"
}

func (r *azureSentinelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *azureSentinelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage an Azure Sentinel integration in Orca. Creates an external service config of `service_name = \"azure_sentinel\"`. The Log Analytics workspace primary key is stored in Orca's secret store and is never returned by the API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Orca external service config identifier (UUID).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"template_name": schema.StringAttribute{
				Required:    true,
				Description: "Template name for the Azure Sentinel integration. Acts as the human-readable identifier for the integration in Orca. Changing this forces a new resource.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"log_type": schema.StringAttribute{
				Required:    true,
				Description: "Custom log type name used in the Log Analytics workspace (for example, `OrcaAlerts`).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"workspace_id": schema.StringAttribute{
				Required:    true,
				Description: "Azure Log Analytics workspace ID that backs the Sentinel instance.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"primary_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Azure Log Analytics workspace primary key. Stored in Orca's secret store; never returned by the API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"is_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the Azure Sentinel integration is enabled. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"is_default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this integration is the organisation's default Azure Sentinel configuration. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
			"business_units": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Optional set of Orca business unit IDs that may use this integration. Leave unset to make the integration available to all business units the caller can access.",
			},
		},
	}
}

func (r *azureSentinelResource) buildPayload(ctx context.Context, plan azureSentinelResourceModel, diags *diag.Diagnostics) api_client.AzureSentinelExternalServiceConfig {
	payload := api_client.AzureSentinelExternalServiceConfig{
		TemplateName: plan.TemplateName.ValueString(),
		IsEnabled:    plan.IsEnabled.ValueBool(),
		IsDefault:    plan.IsDefault.ValueBool(),
		Config: api_client.AzureSentinelConfig{
			LogType:     plan.LogType.ValueString(),
			PrimaryKey:  plan.PrimaryKey.ValueString(),
			WorkspaceID: plan.WorkspaceID.ValueString(),
		},
	}
	payload.BusinessUnits = common.BusinessUnitsToAPI(ctx, plan.BusinessUnits, diags)
	return payload
}

func (r *azureSentinelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating Azure Sentinel integration", "API client not configured.")
		return
	}

	var plan azureSentinelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := r.buildPayload(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.apiClient.CreateAzureSentinelConfig(payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Azure Sentinel integration",
			fmt.Sprintf("Could not create Azure Sentinel integration: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.IsEnabled = types.BoolValue(created.IsEnabled)
	plan.IsDefault = types.BoolValue(created.IsDefault)
	if created.TemplateName != "" {
		plan.TemplateName = types.StringValue(created.TemplateName)
	}
	if created.Config.LogType != "" {
		plan.LogType = types.StringValue(created.Config.LogType)
	}
	if created.Config.WorkspaceID != "" {
		plan.WorkspaceID = types.StringValue(created.Config.WorkspaceID)
	}
	// primary_key is SSM-backed and stripped from responses; keep the planned value.
	bus, busDiags := common.BusinessUnitsFromAPI(ctx, created.BusinessUnits, plan.BusinessUnits)
	resp.Diagnostics.Append(busDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.BusinessUnits = bus

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *azureSentinelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state azureSentinelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.apiClient.GetAzureSentinelConfig(state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Azure Sentinel integration",
			fmt.Sprintf("Could not read Azure Sentinel integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	if current == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(current.ID)
	state.TemplateName = types.StringValue(current.TemplateName)
	state.IsEnabled = types.BoolValue(current.IsEnabled)
	state.IsDefault = types.BoolValue(current.IsDefault)
	if current.Config.LogType != "" {
		state.LogType = types.StringValue(current.Config.LogType)
	}
	if current.Config.WorkspaceID != "" {
		state.WorkspaceID = types.StringValue(current.Config.WorkspaceID)
	}
	bus, busDiags := common.BusinessUnitsFromAPI(ctx, current.BusinessUnits, state.BusinessUnits)
	resp.Diagnostics.Append(busDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.BusinessUnits = bus
	// Orca strips primary_key from responses (stored in SSM); keep existing state value.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *azureSentinelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan azureSentinelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state azureSentinelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := r.buildPayload(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.apiClient.UpdateAzureSentinelConfig(state.TemplateName.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Azure Sentinel integration",
			fmt.Sprintf("Could not update Azure Sentinel integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(updated.ID)
	plan.IsEnabled = types.BoolValue(updated.IsEnabled)
	plan.IsDefault = types.BoolValue(updated.IsDefault)
	if updated.TemplateName != "" {
		plan.TemplateName = types.StringValue(updated.TemplateName)
	}
	if updated.Config.LogType != "" {
		plan.LogType = types.StringValue(updated.Config.LogType)
	}
	if updated.Config.WorkspaceID != "" {
		plan.WorkspaceID = types.StringValue(updated.Config.WorkspaceID)
	}
	bus, busDiags := common.BusinessUnitsFromAPI(ctx, updated.BusinessUnits, plan.BusinessUnits)
	resp.Diagnostics.Append(busDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.BusinessUnits = bus

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *azureSentinelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state azureSentinelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteAzureSentinelConfig(state.TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Azure Sentinel integration",
			fmt.Sprintf("Could not delete Azure Sentinel integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *azureSentinelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
