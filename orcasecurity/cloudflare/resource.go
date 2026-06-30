package cloudflare

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
	_ resource.Resource                = &cloudflareResource{}
	_ resource.ResourceWithConfigure   = &cloudflareResource{}
	_ resource.ResourceWithImportState = &cloudflareResource{}
)

type cloudflareResource struct {
	apiClient *api_client.APIClient
}

type cloudflareResourceModel struct {
	ID           types.String `tfsdk:"id"`
	TemplateName types.String `tfsdk:"template_name"`
	APIToken     types.String `tfsdk:"api_token"`
	IsEnabled    types.Bool   `tfsdk:"is_enabled"`
	IsDefault    types.Bool   `tfsdk:"is_default"`
}

func NewCloudflareResource() resource.Resource {
	return &cloudflareResource{}
}

func (r *cloudflareResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_cloudflare"
}

func (r *cloudflareResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *cloudflareResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage a Cloudflare integration in Orca. Creates an external service config of `service_name = \"cloudflare\"`. The API token is stored in Orca's secret store and is never returned by the API.",
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
				Description: "Template name for the Cloudflare integration. Acts as the human-readable identifier for the integration in Orca. Changing this forces a new resource.",
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
				Description: "Cloudflare API token. Stored in Orca's secret store; never returned by the API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"is_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the Cloudflare integration is enabled. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"is_default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this integration is the organisation's default Cloudflare configuration. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *cloudflareResource) buildPayload(plan cloudflareResourceModel) api_client.CloudflareExternalServiceConfig {
	return api_client.CloudflareExternalServiceConfig{
		TemplateName: plan.TemplateName.ValueString(),
		IsEnabled:    plan.IsEnabled.ValueBool(),
		IsDefault:    plan.IsDefault.ValueBool(),
		Config: api_client.CloudflareConfig{
			APIToken: plan.APIToken.ValueString(),
		},
	}
}

func (r *cloudflareResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating Cloudflare integration", "API client not configured.")
		return
	}

	var plan cloudflareResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.apiClient.CreateCloudflareConfig(r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Cloudflare integration",
			fmt.Sprintf("Could not create Cloudflare integration: %s", err.Error()),
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

func (r *cloudflareResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cloudflareResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.apiClient.GetCloudflareConfig(state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Cloudflare integration",
			fmt.Sprintf("Could not read Cloudflare integration %s: %s", state.TemplateName.ValueString(), err.Error()),
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
	// Orca strips the API token from responses (stored in SSM); keep existing state value.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *cloudflareResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cloudflareResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state cloudflareResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.apiClient.UpdateCloudflareConfig(state.TemplateName.ValueString(), r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Cloudflare integration",
			fmt.Sprintf("Could not update Cloudflare integration %s: %s", state.TemplateName.ValueString(), err.Error()),
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

func (r *cloudflareResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cloudflareResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteCloudflareConfig(state.TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Cloudflare integration",
			fmt.Sprintf("Could not delete Cloudflare integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *cloudflareResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
