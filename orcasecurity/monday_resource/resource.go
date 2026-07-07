package monday_resource

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
	_ resource.Resource                = &mondayResource{}
	_ resource.ResourceWithConfigure   = &mondayResource{}
	_ resource.ResourceWithImportState = &mondayResource{}
)

type mondayResource struct {
	apiClient *api_client.APIClient
}

type mondayResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	APIToken    types.String `tfsdk:"api_token"`
	AccountSlug types.String `tfsdk:"account_slug"`
}

func NewMondayResource() resource.Resource {
	return &mondayResource{}
}

func (r *mondayResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_monday_resource"
}

func (r *mondayResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *mondayResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage a Monday.com credentials resource in Orca. Creates an external service resource of `service_name = \"monday\"` from a Monday API token. Use the returned `id` as `resource_id` on `orcasecurity_integration_monday_template`.",
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
				Description: "Human-friendly name for the Monday.com resource.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"api_token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Monday.com API token. The value is stored in Orca's secret store and is never returned by the API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"account_slug": schema.StringAttribute{
				Computed:    true,
				Description: "Monday.com account slug Orca derives from the API token.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *mondayResource) buildPayload(plan mondayResourceModel) api_client.MondayResource {
	token := plan.APIToken.ValueString()
	return api_client.MondayResource{
		Name: plan.Name.ValueString(),
		Data: api_client.MondayResourceData{
			APIToken: &token,
		},
	}
}

// applyResponse copies API-returned fields onto the plan/state, preserving the write-only
// api_token (the API never returns it).
func applyResponse(m *mondayResourceModel, api *api_client.MondayResource) {
	m.ID = types.StringValue(api.ID)
	if api.Name != "" {
		m.Name = types.StringValue(api.Name)
	}
	m.AccountSlug = types.StringValue(api.Data.AccountSlug)
}

func (r *mondayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating Monday resource", "API client not configured.")
		return
	}

	var plan mondayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.apiClient.CreateMondayResource(r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Monday resource",
			fmt.Sprintf("Could not create Monday resource: %s", err.Error()),
		)
		return
	}

	applyResponse(&plan, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *mondayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mondayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.apiClient.GetMondayResource(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Monday resource",
			fmt.Sprintf("Could not read Monday resource %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}
	if current == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// api_token is preserved from state — the Orca API strips it from responses (SSM-backed).
	if current.Name != "" {
		state.Name = types.StringValue(current.Name)
	}
	state.AccountSlug = types.StringValue(current.Data.AccountSlug)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *mondayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mondayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state mondayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.apiClient.UpdateMondayResource(state.ID.ValueString(), r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Monday resource",
			fmt.Sprintf("Could not update Monday resource %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	applyResponse(&plan, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *mondayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mondayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteMondayResource(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Monday resource",
			fmt.Sprintf("Could not delete Monday resource %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}
}

func (r *mondayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
