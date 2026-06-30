package opsgenie

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

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
	_ resource.Resource                = &opsgenieResource{}
	_ resource.ResourceWithConfigure   = &opsgenieResource{}
	_ resource.ResourceWithImportState = &opsgenieResource{}
)

type opsgenieResource struct {
	apiClient *api_client.APIClient
}

type opsgenieResourceModel struct {
	ID            types.String `tfsdk:"id"`
	TemplateName  types.String `tfsdk:"template_name"`
	OpsgenieKey   types.String `tfsdk:"opsgenie_key"`
	IsEnabled     types.Bool   `tfsdk:"is_enabled"`
	IsDefault     types.Bool   `tfsdk:"is_default"`
	BusinessUnits types.Set    `tfsdk:"business_units"`
}

func NewOpsgenieResource() resource.Resource {
	return &opsgenieResource{}
}

func (r *opsgenieResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_opsgenie"
}

func (r *opsgenieResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *opsgenieResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage an Opsgenie integration in Orca. Creates an external service config of `service_name = \"opsgenie\"` and stores the Opsgenie API key in Orca's secret store.",
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
				Description: "Template name for the Opsgenie integration. Acts as the human-readable identifier for the integration in Orca. Changing this forces a new resource.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"opsgenie_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Opsgenie API integration key. The value is stored in Orca's secret store and is never returned by the API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"is_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the Opsgenie integration is enabled. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"is_default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this integration is the organisation's default Opsgenie configuration. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
			"business_units": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Optional set of Orca business unit IDs that may use this integration. Order does not matter. Leave unset to make the integration available to all business units the caller can access.",
			},
		},
	}
}

func (r *opsgenieResource) buildPayload(ctx context.Context, plan opsgenieResourceModel, diags *diag.Diagnostics) api_client.OpsgenieExternalServiceConfig {
	payload := api_client.OpsgenieExternalServiceConfig{
		TemplateName: plan.TemplateName.ValueString(),
		IsEnabled:    plan.IsEnabled.ValueBool(),
		IsDefault:    plan.IsDefault.ValueBool(),
		Config: api_client.OpsgenieConfig{
			OpsgenieKey: plan.OpsgenieKey.ValueString(),
		},
	}
	if !plan.BusinessUnits.IsNull() && !plan.BusinessUnits.IsUnknown() {
		var bus []string
		diags.Append(plan.BusinessUnits.ElementsAs(ctx, &bus, false)...)
		payload.BusinessUnits = bus
	}
	return payload
}

// businessUnitsFromAPI converts the API's business_units field into a Terraform list value.
// When the user did not declare business_units in their config (planned value is null) and the
// API returns an empty list, preserve the null to avoid a "null vs []" diff on every plan.
func businessUnitsFromAPI(ctx context.Context, apiBus []string, planned types.Set) (types.Set, diag.Diagnostics) {
	if len(apiBus) == 0 && planned.IsNull() {
		return types.SetNull(types.StringType), nil
	}
	return types.SetValueFrom(ctx, types.StringType, apiBus)
}

func (r *opsgenieResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating Opsgenie integration", "API client not configured.")
		return
	}

	var plan opsgenieResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := r.buildPayload(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.apiClient.CreateOpsgenieConfig(payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Opsgenie integration",
			fmt.Sprintf("Could not create Opsgenie integration: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.IsEnabled = types.BoolValue(created.IsEnabled)
	plan.IsDefault = types.BoolValue(created.IsDefault)
	if created.TemplateName != "" {
		plan.TemplateName = types.StringValue(created.TemplateName)
	}
	bus, busDiags := businessUnitsFromAPI(ctx, created.BusinessUnits, plan.BusinessUnits)
	resp.Diagnostics.Append(busDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.BusinessUnits = bus

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *opsgenieResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state opsgenieResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.apiClient.GetOpsgenieConfig(state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Opsgenie integration",
			fmt.Sprintf("Could not read Opsgenie integration %s: %s", state.TemplateName.ValueString(), err.Error()),
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
	bus, busDiags := businessUnitsFromAPI(ctx, current.BusinessUnits, state.BusinessUnits)
	resp.Diagnostics.Append(busDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.BusinessUnits = bus
	// Orca strips the Opsgenie key from responses (stored in SSM); keep existing state value.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *opsgenieResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan opsgenieResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state opsgenieResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := r.buildPayload(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.apiClient.UpdateOpsgenieConfig(state.TemplateName.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Opsgenie integration",
			fmt.Sprintf("Could not update Opsgenie integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(updated.ID)
	plan.IsEnabled = types.BoolValue(updated.IsEnabled)
	plan.IsDefault = types.BoolValue(updated.IsDefault)
	if updated.TemplateName != "" {
		plan.TemplateName = types.StringValue(updated.TemplateName)
	}
	bus, busDiags := businessUnitsFromAPI(ctx, updated.BusinessUnits, plan.BusinessUnits)
	resp.Diagnostics.Append(busDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.BusinessUnits = bus

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *opsgenieResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state opsgenieResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteOpsgenieConfig(state.TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Opsgenie integration",
			fmt.Sprintf("Could not delete Opsgenie integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *opsgenieResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Config endpoints look up integrations by template_name; import keys on that value.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
