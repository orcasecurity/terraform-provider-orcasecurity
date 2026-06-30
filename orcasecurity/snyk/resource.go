package snyk

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
	_ resource.Resource                = &snykResource{}
	_ resource.ResourceWithConfigure   = &snykResource{}
	_ resource.ResourceWithImportState = &snykResource{}
)

// Allowed Snyk region codes. Mirrors the JSON schema in
// base_api/api/external_services/schemas/snyk.schema.json.
var snykRegions = []string{"US", "EU", "AU", "US2"}

type snykResource struct {
	apiClient *api_client.APIClient
}

type snykResourceModel struct {
	ID           types.String `tfsdk:"id"`
	TemplateName types.String `tfsdk:"template_name"`
	APIToken     types.String `tfsdk:"api_token"`
	Region       types.String `tfsdk:"region"`
	IsEnabled    types.Bool   `tfsdk:"is_enabled"`
	IsDefault    types.Bool   `tfsdk:"is_default"`
}

func NewSnykResource() resource.Resource {
	return &snykResource{}
}

func (r *snykResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_snyk"
}

func (r *snykResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *snykResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage a Snyk integration in Orca. Creates an external service config of `service_name = \"snyk\"`. The Snyk API token is stored in Orca's secret store and is never returned by the API.",
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
				Description: "Template name for the Snyk integration. Acts as the human-readable identifier for the integration in Orca. Changing this forces a new resource.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"api_token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Snyk service account API token. Stored in Orca's secret store; never returned by the API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"region": schema.StringAttribute{
				Required:    true,
				Description: "Snyk tenant region. The Orca UI exposes four choices — pass the matching API code here:\n" +
					"  - `US`  — United States (app.snyk.io)\n" +
					"  - `US2` — United States 2 (app.us.snyk.io)\n" +
					"  - `EU`  — European Union (app.eu.snyk.io)\n" +
					"  - `AU`  — Australia (app.au.snyk.io)",
				Validators: []validator.String{
					stringvalidator.OneOf(snykRegions...),
				},
			},
			"is_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the Snyk integration is enabled. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"is_default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this integration is the organisation's default Snyk configuration. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *snykResource) buildPayload(plan snykResourceModel) api_client.SnykExternalServiceConfig {
	return api_client.SnykExternalServiceConfig{
		TemplateName: plan.TemplateName.ValueString(),
		IsEnabled:    plan.IsEnabled.ValueBool(),
		IsDefault:    plan.IsDefault.ValueBool(),
		Config: api_client.SnykConfig{
			APIToken: plan.APIToken.ValueString(),
			Region:   plan.Region.ValueString(),
		},
	}
}

func (r *snykResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating Snyk integration", "API client not configured.")
		return
	}

	var plan snykResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.apiClient.CreateSnykConfig(r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Snyk integration",
			fmt.Sprintf("Could not create Snyk integration: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.IsEnabled = types.BoolValue(created.IsEnabled)
	plan.IsDefault = types.BoolValue(created.IsDefault)
	if created.TemplateName != "" {
		plan.TemplateName = types.StringValue(created.TemplateName)
	}
	if created.Config.Region != "" {
		plan.Region = types.StringValue(created.Config.Region)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *snykResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state snykResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.apiClient.GetSnykConfig(state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Snyk integration",
			fmt.Sprintf("Could not read Snyk integration %s: %s", state.TemplateName.ValueString(), err.Error()),
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
	if current.Config.Region != "" {
		state.Region = types.StringValue(current.Config.Region)
	}
	// Orca strips api_token from responses (stored in SSM); keep existing state value.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *snykResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan snykResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state snykResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.apiClient.UpdateSnykConfig(state.TemplateName.ValueString(), r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Snyk integration",
			fmt.Sprintf("Could not update Snyk integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(updated.ID)
	plan.IsEnabled = types.BoolValue(updated.IsEnabled)
	plan.IsDefault = types.BoolValue(updated.IsDefault)
	if updated.TemplateName != "" {
		plan.TemplateName = types.StringValue(updated.TemplateName)
	}
	if updated.Config.Region != "" {
		plan.Region = types.StringValue(updated.Config.Region)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *snykResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state snykResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteSnykConfig(state.TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Snyk integration",
			fmt.Sprintf("Could not delete Snyk integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *snykResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
