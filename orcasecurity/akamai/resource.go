package akamai

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
	_ resource.Resource                = &akamaiResource{}
	_ resource.ResourceWithConfigure   = &akamaiResource{}
	_ resource.ResourceWithImportState = &akamaiResource{}
)

type akamaiResource struct {
	apiClient *api_client.APIClient
}

type akamaiResourceModel struct {
	ID           types.String `tfsdk:"id"`
	TemplateName types.String `tfsdk:"template_name"`
	AccessToken  types.String `tfsdk:"access_token"`
	ClientToken  types.String `tfsdk:"client_token"`
	ClientSecret types.String `tfsdk:"client_secret"`
	Host         types.String `tfsdk:"host"`
	IsEnabled    types.Bool   `tfsdk:"is_enabled"`
	IsDefault    types.Bool   `tfsdk:"is_default"`
}

func NewAkamaiResource() resource.Resource {
	return &akamaiResource{}
}

func (r *akamaiResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_akamai"
}

func (r *akamaiResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *akamaiResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage an Akamai integration in Orca. Creates an external service config of `service_name = \"akamai\"`. The Akamai credentials (`access_token`, `client_token`, `client_secret`) are stored in Orca's secret store and are never returned by the API.",
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
				Description: "Template name for the Akamai integration. Acts as the human-readable identifier for the integration in Orca. Changing this forces a new resource.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access_token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Akamai EdgeGrid `access_token`. Stored in Orca's secret store; never returned by the API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"client_token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Akamai EdgeGrid `client_token`. Stored in Orca's secret store; never returned by the API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"client_secret": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Akamai EdgeGrid `client_secret`. Stored in Orca's secret store; never returned by the API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"host": schema.StringAttribute{
				Required:    true,
				Description: "Akamai EdgeGrid host (for example, `akab-xxxxxxxx.luna.akamaiapis.net`).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"is_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the Akamai integration is enabled. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"is_default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this integration is the organisation's default Akamai configuration. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *akamaiResource) buildPayload(plan akamaiResourceModel) api_client.AkamaiExternalServiceConfig {
	return api_client.AkamaiExternalServiceConfig{
		TemplateName: plan.TemplateName.ValueString(),
		IsEnabled:    plan.IsEnabled.ValueBool(),
		IsDefault:    plan.IsDefault.ValueBool(),
		Config: api_client.AkamaiConfig{
			AccessToken:  plan.AccessToken.ValueString(),
			ClientToken:  plan.ClientToken.ValueString(),
			ClientSecret: plan.ClientSecret.ValueString(),
			Host:         plan.Host.ValueString(),
		},
	}
}

func (r *akamaiResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating Akamai integration", "API client not configured.")
		return
	}

	var plan akamaiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.apiClient.CreateAkamaiConfig(r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Akamai integration",
			fmt.Sprintf("Could not create Akamai integration: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.IsEnabled = types.BoolValue(created.IsEnabled)
	plan.IsDefault = types.BoolValue(created.IsDefault)
	if created.TemplateName != "" {
		plan.TemplateName = types.StringValue(created.TemplateName)
	}
	// Keep secrets exactly as the user planned them; the API strips access_token, client_token,
	// and client_secret from responses. Refresh host from the response since it is not secret.
	if created.Config.Host != "" {
		plan.Host = types.StringValue(created.Config.Host)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *akamaiResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state akamaiResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.apiClient.GetAkamaiConfig(state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Akamai integration",
			fmt.Sprintf("Could not read Akamai integration %s: %s", state.TemplateName.ValueString(), err.Error()),
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
	if current.Config.Host != "" {
		state.Host = types.StringValue(current.Config.Host)
	}
	// Orca strips Akamai secrets from responses (stored in SSM); keep existing state values.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *akamaiResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan akamaiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state akamaiResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.apiClient.UpdateAkamaiConfig(state.TemplateName.ValueString(), r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Akamai integration",
			fmt.Sprintf("Could not update Akamai integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(updated.ID)
	plan.IsEnabled = types.BoolValue(updated.IsEnabled)
	plan.IsDefault = types.BoolValue(updated.IsDefault)
	if updated.TemplateName != "" {
		plan.TemplateName = types.StringValue(updated.TemplateName)
	}
	if updated.Config.Host != "" {
		plan.Host = types.StringValue(updated.Config.Host)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *akamaiResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state akamaiResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteAkamaiConfig(state.TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Akamai integration",
			fmt.Sprintf("Could not delete Akamai integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *akamaiResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
