package servicenow

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &serviceNowITSMResource{}
	_ resource.ResourceWithConfigure   = &serviceNowITSMResource{}
	_ resource.ResourceWithImportState = &serviceNowITSMResource{}
)

type serviceNowITSMResource struct {
	apiClient *api_client.APIClient
}

type serviceNowITSMResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	URL      types.String `tfsdk:"servicenow_url"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func NewServiceNowResource() resource.Resource {
	return &serviceNowITSMResource{}
}

func (r *serviceNowITSMResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_servicenow_resource"
}

func (r *serviceNowITSMResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *serviceNowITSMResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage a ServiceNow ITSM integration in Orca. Creates an external service resource of type `sn_incidents` using Basic authentication.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Orca resource identifier (UUID).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-friendly name for the ServiceNow ITSM integration.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"servicenow_url": schema.StringAttribute{
				Required:    true,
				Description: "ServiceNow instance URL (for example, `https://my-instance.service-now.com`).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "ServiceNow account username used for Basic authentication.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"password": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "ServiceNow account password used for Basic authentication. The value is stored in Orca's secret store and is never returned by the API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *serviceNowITSMResource) buildPayload(plan serviceNowITSMResourceModel) api_client.ServiceNowITSMResource {
	password := plan.Password.ValueString()
	return api_client.ServiceNowITSMResource{
		Name:    plan.Name.ValueString(),
		HostURL: plan.URL.ValueString(),
		Data: api_client.ServiceNowITSMData{
			Username: plan.Username.ValueString(),
			Password: &password,
		},
	}
}

func (r *serviceNowITSMResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating ServiceNow ITSM integration", "API client not configured.")
		return
	}

	var plan serviceNowITSMResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.apiClient.CreateServiceNowITSMResource(r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating ServiceNow ITSM integration",
			fmt.Sprintf("Could not create ServiceNow ITSM integration: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(created.ID)
	// Keep the user-supplied URL/name in state in case the API normalises them. Refresh from
	// the API response when it returns a value to avoid spurious diffs.
	if created.HostURL != "" {
		plan.URL = types.StringValue(created.HostURL)
	}
	if created.Name != "" {
		plan.Name = types.StringValue(created.Name)
	}
	if created.Data.Username != "" {
		plan.Username = types.StringValue(created.Data.Username)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceNowITSMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceNowITSMResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.apiClient.GetServiceNowITSMResource(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading ServiceNow ITSM integration",
			fmt.Sprintf("Could not read ServiceNow ITSM integration %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	if current == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(current.Name)
	state.URL = types.StringValue(current.HostURL)
	if current.Data.Username != "" {
		state.Username = types.StringValue(current.Data.Username)
	}
	// The Orca API strips the password from responses (the value lives in SSM).
	// Preserve whatever the user already has in state so Terraform does not flag a drift.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serviceNowITSMResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceNowITSMResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state serviceNowITSMResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.apiClient.UpdateServiceNowITSMResource(state.ID.ValueString(), r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating ServiceNow ITSM integration",
			fmt.Sprintf("Could not update ServiceNow ITSM integration %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(updated.ID)
	if updated.HostURL != "" {
		plan.URL = types.StringValue(updated.HostURL)
	}
	if updated.Name != "" {
		plan.Name = types.StringValue(updated.Name)
	}
	if updated.Data.Username != "" {
		plan.Username = types.StringValue(updated.Data.Username)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceNowITSMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceNowITSMResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteServiceNowITSMResource(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting ServiceNow ITSM integration",
			fmt.Sprintf("Could not delete ServiceNow ITSM integration %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}
}

func (r *serviceNowITSMResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
