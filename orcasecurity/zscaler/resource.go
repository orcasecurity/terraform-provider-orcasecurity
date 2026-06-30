package zscaler

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
	_ resource.Resource                = &zscalerResource{}
	_ resource.ResourceWithConfigure   = &zscalerResource{}
	_ resource.ResourceWithImportState = &zscalerResource{}
)

type zscalerResource struct {
	apiClient *api_client.APIClient
}

type zscalerResourceModel struct {
	ID           types.String `tfsdk:"id"`
	TemplateName types.String `tfsdk:"template_name"`
	VanityDomain types.String `tfsdk:"vanity_domain"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	IsEnabled    types.Bool   `tfsdk:"is_enabled"`
	IsDefault    types.Bool   `tfsdk:"is_default"`
}

func NewZscalerResource() resource.Resource {
	return &zscalerResource{}
}

func (r *zscalerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_zscaler_zpa"
}

func (r *zscalerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *zscalerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage a Zscaler ZPA integration in Orca. Creates an external service config of `service_name = \"zscaler\"`. The OAuth `client_id` and `client_secret` are stored in Orca's secret store and are never returned by the API.",
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
				Description: "Template name for the Zscaler ZPA integration. Acts as the human-readable identifier for the integration in Orca. Changing this forces a new resource.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vanity_domain": schema.StringAttribute{
				Required:    true,
				Description: "Zscaler ZPA vanity domain (the customer-specific tenant identifier used in the OAuth URL).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"client_id": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Zscaler ZPA OAuth `client_id`. Stored in Orca's secret store; never returned by the API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"client_secret": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Zscaler ZPA OAuth `client_secret`. Stored in Orca's secret store; never returned by the API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"is_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the Zscaler ZPA integration is enabled. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"is_default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this integration is the organisation's default Zscaler ZPA configuration. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *zscalerResource) buildPayload(plan zscalerResourceModel) api_client.ZscalerExternalServiceConfig {
	return api_client.ZscalerExternalServiceConfig{
		TemplateName: plan.TemplateName.ValueString(),
		IsEnabled:    plan.IsEnabled.ValueBool(),
		IsDefault:    plan.IsDefault.ValueBool(),
		Config: api_client.ZscalerConfig{
			VanityDomain: plan.VanityDomain.ValueString(),
			ClientID:     plan.ClientID.ValueString(),
			ClientSecret: plan.ClientSecret.ValueString(),
		},
	}
}

func (r *zscalerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating Zscaler ZPA integration", "API client not configured.")
		return
	}

	var plan zscalerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.apiClient.CreateZscalerConfig(r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Zscaler ZPA integration",
			fmt.Sprintf("Could not create Zscaler ZPA integration: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.IsEnabled = types.BoolValue(created.IsEnabled)
	plan.IsDefault = types.BoolValue(created.IsDefault)
	if created.TemplateName != "" {
		plan.TemplateName = types.StringValue(created.TemplateName)
	}
	// Refresh vanity_domain from the response (non-secret); leave client_id and client_secret
	// as the user planned them since the API strips them.
	if created.Config.VanityDomain != "" {
		plan.VanityDomain = types.StringValue(created.Config.VanityDomain)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *zscalerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state zscalerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.apiClient.GetZscalerConfig(state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Zscaler ZPA integration",
			fmt.Sprintf("Could not read Zscaler ZPA integration %s: %s", state.TemplateName.ValueString(), err.Error()),
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
	if current.Config.VanityDomain != "" {
		state.VanityDomain = types.StringValue(current.Config.VanityDomain)
	}
	// Orca strips client_id and client_secret from responses (stored in SSM); keep existing
	// state values.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *zscalerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan zscalerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state zscalerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.apiClient.UpdateZscalerConfig(state.TemplateName.ValueString(), r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Zscaler ZPA integration",
			fmt.Sprintf("Could not update Zscaler ZPA integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(updated.ID)
	plan.IsEnabled = types.BoolValue(updated.IsEnabled)
	plan.IsDefault = types.BoolValue(updated.IsDefault)
	if updated.TemplateName != "" {
		plan.TemplateName = types.StringValue(updated.TemplateName)
	}
	if updated.Config.VanityDomain != "" {
		plan.VanityDomain = types.StringValue(updated.Config.VanityDomain)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *zscalerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state zscalerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteZscalerConfig(state.TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Zscaler ZPA integration",
			fmt.Sprintf("Could not delete Zscaler ZPA integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *zscalerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
