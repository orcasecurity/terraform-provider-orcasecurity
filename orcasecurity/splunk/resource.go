package splunk

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
	_ resource.Resource                = &splunkResource{}
	_ resource.ResourceWithConfigure   = &splunkResource{}
	_ resource.ResourceWithImportState = &splunkResource{}
)

type splunkResource struct {
	apiClient *api_client.APIClient
}

type splunkResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	TemplateName        types.String `tfsdk:"template_name"`
	URL                 types.String `tfsdk:"url"`
	Token               types.String `tfsdk:"token"`
	AllowSelfSignedCert types.Bool   `tfsdk:"allow_self_signed_cert"`
	IsEnabled           types.Bool   `tfsdk:"is_enabled"`
	IsDefault           types.Bool   `tfsdk:"is_default"`
}

func NewSplunkResource() resource.Resource {
	return &splunkResource{}
}

func (r *splunkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_splunk"
}

func (r *splunkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *splunkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage a Splunk HEC integration in Orca. Creates an external service config of `service_name = \"splunk\"`. The HEC token is stored in Orca's secret store and is never returned by the API.",
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
				Description: "Template name for the Splunk integration. Acts as the human-readable identifier for the integration in Orca. Changing this forces a new resource.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"url": schema.StringAttribute{
				Required:    true,
				Description: "Splunk HEC endpoint URL (for example, `https://prd-p-xxxxx.splunkcloud.com:8088/services/collector/event`).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Splunk HEC token. Stored in Orca's secret store; never returned by the API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"allow_self_signed_cert": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether Orca should accept self-signed TLS certificates when calling the Splunk endpoint. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
			"is_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the Splunk integration is enabled. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"is_default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this integration is the organisation's default Splunk configuration. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *splunkResource) buildPayload(plan splunkResourceModel) api_client.SplunkExternalServiceConfig {
	return api_client.SplunkExternalServiceConfig{
		TemplateName: plan.TemplateName.ValueString(),
		IsEnabled:    plan.IsEnabled.ValueBool(),
		IsDefault:    plan.IsDefault.ValueBool(),
		Config: api_client.SplunkConfig{
			URL:                 plan.URL.ValueString(),
			Token:               plan.Token.ValueString(),
			AllowSelfSignedCert: plan.AllowSelfSignedCert.ValueBool(),
		},
	}
}

func (r *splunkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating Splunk integration", "API client not configured.")
		return
	}

	var plan splunkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.apiClient.CreateSplunkConfig(r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Splunk integration",
			fmt.Sprintf("Could not create Splunk integration: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.IsEnabled = types.BoolValue(created.IsEnabled)
	plan.IsDefault = types.BoolValue(created.IsDefault)
	if created.TemplateName != "" {
		plan.TemplateName = types.StringValue(created.TemplateName)
	}
	if created.Config.URL != "" {
		plan.URL = types.StringValue(created.Config.URL)
	}
	plan.AllowSelfSignedCert = types.BoolValue(created.Config.AllowSelfSignedCert)
	// token is SSM-backed and stripped from the response; keep the planned value.

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *splunkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state splunkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.apiClient.GetSplunkConfig(state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Splunk integration",
			fmt.Sprintf("Could not read Splunk integration %s: %s", state.TemplateName.ValueString(), err.Error()),
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
	if current.Config.URL != "" {
		state.URL = types.StringValue(current.Config.URL)
	}
	state.AllowSelfSignedCert = types.BoolValue(current.Config.AllowSelfSignedCert)
	// Orca strips the token from responses (stored in SSM); keep existing state value.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *splunkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan splunkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state splunkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.apiClient.UpdateSplunkConfig(state.TemplateName.ValueString(), r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Splunk integration",
			fmt.Sprintf("Could not update Splunk integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(updated.ID)
	plan.IsEnabled = types.BoolValue(updated.IsEnabled)
	plan.IsDefault = types.BoolValue(updated.IsDefault)
	if updated.TemplateName != "" {
		plan.TemplateName = types.StringValue(updated.TemplateName)
	}
	if updated.Config.URL != "" {
		plan.URL = types.StringValue(updated.Config.URL)
	}
	plan.AllowSelfSignedCert = types.BoolValue(updated.Config.AllowSelfSignedCert)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *splunkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state splunkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteSplunkConfig(state.TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Splunk integration",
			fmt.Sprintf("Could not delete Splunk integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *splunkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
