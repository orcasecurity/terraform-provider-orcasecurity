package discovery_view

import (
	"context"
	"encoding/json"
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
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &discoveryViewResource{}
	_ resource.ResourceWithConfigure   = &discoveryViewResource{}
	_ resource.ResourceWithImportState = &discoveryViewResource{}
)

type discoveryViewResource struct {
	apiClient *api_client.APIClient
}

type discoveryQueryResourceModel struct {
	Data types.String `tfsdk:"query"`
}

type discoveryViewResourceModel struct {
	ID                types.String                `tfsdk:"id"`
	Name              types.String                `tfsdk:"name"`
	FilterData        discoveryQueryResourceModel `tfsdk:"filter_data"`
	ExtraParameters   map[string]interface{}      `tfsdk:"extra_params"`
	OrganizationLevel types.Bool                  `tfsdk:"organization_level"`
	ViewType          types.String                `tfsdk:"view_type"`
}

func NewDiscoveryViewResource() resource.Resource {
	return &discoveryViewResource{}
}

func (r *discoveryViewResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_discovery_view"
}

func (r *discoveryViewResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *discoveryViewResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *discoveryViewResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	//tflog.Error(ctx, "Setting up Schema")
	resp.Schema = schema.Schema{
		Description: "Provides a Discovery view resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Discovery view ID.",
			},
			"name": schema.StringAttribute{
				Description: "Discovery view name.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"view_type": schema.StringAttribute{
				Description: "Should be set to 'discovery' for discovery views.",
				Required:    true,
			},
			"organization_level": schema.BoolAttribute{
				Description: "If set to true, it is is a shared discovery view (can be viewed by any member of your Orca org). If set to false, it is a personal discovery view (can be viewed only by you, not other members of your Orca org).",
				Required:    true,
			},
			"extra_params": schema.MapAttribute{
				Description: "NOT YET IMPLEMENTED. Allows you to set a limit of returned results, amongst other things.",
				ElementType: types.StringType,
				Required:    true,
			},
			"filter_data": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"query": schema.StringAttribute{
						Description: "Discovery query that will be created. Should be in JSON format.",
						Required:    true,
					},
				},
			},
		},
	}
}

func (r *discoveryViewResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan discoveryViewResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	//Generate API request body from plan
	queryString := plan.FilterData.Data.ValueString()
	query := make(map[string]interface{})
	err := json.Unmarshal([]byte(queryString), &query)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating discovery view",
			"Could not create discovery view, unexpected error: "+err.Error(),
		)
		return
	}

	createReq := api_client.DiscoveryView{
		Name:              plan.Name.ValueString(),
		OrganizationLevel: plan.OrganizationLevel.ValueBool(),
		ViewType:          plan.ViewType.String()[1 : len(plan.ViewType.String())-1],
		ExtraParameters:   make(map[string]interface{}),
		FilterData:        api_client.DiscoveryQuery{Data: query},
	}

	instance, err := r.apiClient.CreateDiscoveryView(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating discovery view",
			"Could not create discovery view, unexpected error: "+err.Error(),
		)
		return
	}

	//plan.OrganizationLevel = types.BoolValue(instance.OrganizationLevel)
	//plan.ViewType = types.StringValue(instance.ViewType)

	instance, err = r.apiClient.GetDiscoveryView(instance.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error refreshing Discovery view",
			"Could not create Discovery view, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *discoveryViewResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state discoveryViewResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.apiClient.DoesDiscoveryViewExist(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading discovery view",
			fmt.Sprintf("Could not read discovery view ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	if !exists {
		tflog.Warn(ctx, fmt.Sprintf("Discovery view %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	instance, err := r.apiClient.GetDiscoveryView(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading discovery view",
			fmt.Sprintf("Could not read discovery view ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	queryString := state.FilterData.Data.ValueString()

	state.ID = types.StringValue(instance.ID)
	state.Name = types.StringValue(instance.Name)
	state.OrganizationLevel = types.BoolValue(instance.OrganizationLevel)
	state.ViewType = types.StringValue(instance.ViewType)
	state.ExtraParameters = make(map[string]interface{})
	state.FilterData = discoveryQueryResourceModel{Data: types.StringValue(queryString)}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *discoveryViewResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan discoveryViewResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.String() == "" {
		resp.Diagnostics.AddError(
			"ID is null",
			"Could not update discovery view, unexpected error: "+plan.ID.ValueString(),
		)
		return
	}

	//Generate API request body from plan
	queryString := plan.FilterData.Data.ValueString()
	query := make(map[string]interface{})
	err := json.Unmarshal([]byte(queryString), &query)

	updateReq := api_client.DiscoveryView{
		ID:                plan.ID.ValueString(),
		Name:              plan.Name.ValueString(),
		OrganizationLevel: plan.OrganizationLevel.ValueBool(),
		ViewType:          plan.ViewType.String()[1 : len(plan.ViewType.String())-1],
		ExtraParameters:   make(map[string]interface{}),
		FilterData:        api_client.DiscoveryQuery{Data: query},
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating discovery view",
			"Could not unmarshal json, unexpected error: "+err.Error(),
		)
		return
	}

	instance, err2 := r.apiClient.UpdateDiscoveryView(updateReq)
	if err2 != nil {
		resp.Diagnostics.AddError(
			"Error updating discovery view",
			"Could not update discovery view, unexpected error: "+err2.Error(),
		)
		return
	}

	instance, err2 = r.apiClient.GetDiscoveryView(plan.ID.ValueString())
	if err2 != nil {
		resp.Diagnostics.AddError(
			"Error updating discovery view",
			"Could not read discovery view, unexpected error: "+err2.Error(),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.OrganizationLevel = types.BoolValue(instance.OrganizationLevel)
	plan.Name = types.StringValue(instance.Name)
	//plan.ExtraParameters = instance.ExtraParameters
	plan.ViewType = types.StringValue(instance.ViewType)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *discoveryViewResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state discoveryViewResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteDiscoveryView(state.ID.String()[1 : len(state.ID.String())-1])
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting discovery view",
			"Could not delete discovery view, unexpected error: "+err.Error(),
		)
		return
	}
}
