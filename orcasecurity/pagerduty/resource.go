package pagerduty

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.Resource                = &pagerDutyResource{}
	_ resource.ResourceWithConfigure   = &pagerDutyResource{}
	_ resource.ResourceWithImportState = &pagerDutyResource{}
)

type pagerDutyResource struct {
	apiClient *api_client.APIClient
}

type pagerDutyResourceModel struct {
	ID             types.String `tfsdk:"id"`
	TemplateName   types.String `tfsdk:"template_name"`
	IntegrationKey types.String `tfsdk:"integration_key"`
	IsEnabled      types.Bool   `tfsdk:"is_enabled"`
	IsDefault      types.Bool   `tfsdk:"is_default"`
}

func NewPagerDutyResource() resource.Resource {
	return &pagerDutyResource{}
}

func (r *pagerDutyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_pagerduty"
}

func (r *pagerDutyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *pagerDutyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage a PagerDuty integration in Orca. Creates an external service config of `service_name = \"pagerduty\"` and stores the integration key in Orca's secret store.",
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
				Description: "Template name for the PagerDuty integration. Acts as the human-readable identifier for the integration in Orca. Changing this forces a new resource.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"integration_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "PagerDuty Events API V2 integration key. The value is stored in Orca's secret store and is never returned by the API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"is_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the PagerDuty integration is enabled. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"is_default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this integration is the organisation's default PagerDuty configuration. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *pagerDutyResource) buildPayload(plan pagerDutyResourceModel) api_client.PagerDutyExternalServiceConfig {
	return api_client.PagerDutyExternalServiceConfig{
		TemplateName: plan.TemplateName.ValueString(),
		IsEnabled:    plan.IsEnabled.ValueBool(),
		IsDefault:    plan.IsDefault.ValueBool(),
		Config: api_client.PagerDutyConfig{
			IntegrationKey: plan.IntegrationKey.ValueString(),
		},
	}
}

func (r *pagerDutyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating PagerDuty integration", "API client not configured.")
		return
	}

	var plan pagerDutyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.apiClient.CreatePagerDutyConfig(r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating PagerDuty integration",
			fmt.Sprintf("Could not create PagerDuty integration: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.IsEnabled = types.BoolValue(created.IsEnabled)
	plan.IsDefault = types.BoolValue(created.IsDefault)
	if created.TemplateName != "" {
		plan.TemplateName = types.StringValue(created.TemplateName)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pagerDutyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pagerDutyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.apiClient.GetPagerDutyConfig(state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading PagerDuty integration",
			fmt.Sprintf("Could not read PagerDuty integration %s: %s", state.TemplateName.ValueString(), err.Error()),
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
	// Orca strips the integration key from responses (stored in SSM); keep existing state value.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pagerDutyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan pagerDutyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state pagerDutyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.apiClient.UpdatePagerDutyConfig(state.TemplateName.ValueString(), r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating PagerDuty integration",
			fmt.Sprintf("Could not update PagerDuty integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(updated.ID)
	plan.IsEnabled = types.BoolValue(updated.IsEnabled)
	plan.IsDefault = types.BoolValue(updated.IsDefault)
	if updated.TemplateName != "" {
		plan.TemplateName = types.StringValue(updated.TemplateName)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pagerDutyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state pagerDutyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeletePagerDutyConfig(state.TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting PagerDuty integration",
			fmt.Sprintf("Could not delete PagerDuty integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *pagerDutyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Config endpoints look up integrations by template_name; import keys on that value.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
