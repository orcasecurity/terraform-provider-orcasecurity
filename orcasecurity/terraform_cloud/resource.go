package terraform_cloud

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
	_ resource.Resource                = &terraformCloudResource{}
	_ resource.ResourceWithConfigure   = &terraformCloudResource{}
	_ resource.ResourceWithImportState = &terraformCloudResource{}
)

type terraformCloudResource struct {
	apiClient *api_client.APIClient
}

type terraformCloudResourceModel struct {
	ID           types.String `tfsdk:"id"`
	TemplateName types.String `tfsdk:"template_name"`
	APIToken     types.String `tfsdk:"api_token"`
	APIURL       types.String `tfsdk:"api_url"`
	IsEnabled    types.Bool   `tfsdk:"is_enabled"`
	IsDefault    types.Bool   `tfsdk:"is_default"`
}

func NewTerraformCloudResource() resource.Resource {
	return &terraformCloudResource{}
}

func (r *terraformCloudResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_terraform_cloud"
}

func (r *terraformCloudResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *terraformCloudResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage a Terraform Cloud (HCP Terraform) integration in Orca. Creates an external service config of `service_name = \"terraform_cloud\"`. The API token is stored in Orca's secret store and is never returned by the API.",
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
				Description: "Template name for the Terraform Cloud integration. Acts as the human-readable identifier for the integration in Orca. Changing this forces a new resource.",
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
				Description: "Terraform Cloud service-account API token. Stored in Orca's secret store; never returned by the API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"api_url": schema.StringAttribute{
				Required:    true,
				Description: "Terraform Cloud API URL. Use `https://app.terraform.io` for HCP Terraform or your Terraform Enterprise hostname.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"is_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the Terraform Cloud integration is enabled. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"is_default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this integration is the organisation's default Terraform Cloud configuration. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *terraformCloudResource) buildPayload(plan terraformCloudResourceModel) api_client.TerraformCloudExternalServiceConfig {
	return api_client.TerraformCloudExternalServiceConfig{
		TemplateName: plan.TemplateName.ValueString(),
		IsEnabled:    plan.IsEnabled.ValueBool(),
		IsDefault:    plan.IsDefault.ValueBool(),
		Config: api_client.TerraformCloudConfig{
			APIToken: plan.APIToken.ValueString(),
			APIURL:   plan.APIURL.ValueString(),
		},
	}
}

func (r *terraformCloudResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating Terraform Cloud integration", "API client not configured.")
		return
	}

	var plan terraformCloudResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.apiClient.CreateTerraformCloudConfig(r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Terraform Cloud integration",
			fmt.Sprintf("Could not create Terraform Cloud integration: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.IsEnabled = types.BoolValue(created.IsEnabled)
	plan.IsDefault = types.BoolValue(created.IsDefault)
	if created.TemplateName != "" {
		plan.TemplateName = types.StringValue(created.TemplateName)
	}
	if created.Config.APIURL != "" {
		plan.APIURL = types.StringValue(created.Config.APIURL)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *terraformCloudResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state terraformCloudResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.apiClient.GetTerraformCloudConfig(state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Terraform Cloud integration",
			fmt.Sprintf("Could not read Terraform Cloud integration %s: %s", state.TemplateName.ValueString(), err.Error()),
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
	if current.Config.APIURL != "" {
		state.APIURL = types.StringValue(current.Config.APIURL)
	}
	// Orca strips api_token from responses (stored in SSM); keep existing state value.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *terraformCloudResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan terraformCloudResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state terraformCloudResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.apiClient.UpdateTerraformCloudConfig(state.TemplateName.ValueString(), r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Terraform Cloud integration",
			fmt.Sprintf("Could not update Terraform Cloud integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(updated.ID)
	plan.IsEnabled = types.BoolValue(updated.IsEnabled)
	plan.IsDefault = types.BoolValue(updated.IsDefault)
	if updated.TemplateName != "" {
		plan.TemplateName = types.StringValue(updated.TemplateName)
	}
	if updated.Config.APIURL != "" {
		plan.APIURL = types.StringValue(updated.Config.APIURL)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *terraformCloudResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state terraformCloudResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteTerraformCloudConfig(state.TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Terraform Cloud integration",
			fmt.Sprintf("Could not delete Terraform Cloud integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *terraformCloudResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
