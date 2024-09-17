package custom_dashboard

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
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &customDashboardResource{}
	_ resource.ResourceWithConfigure   = &customDashboardResource{}
	_ resource.ResourceWithImportState = &customDashboardResource{}
)

type customDashboardResource struct {
	apiClient *api_client.APIClient
}

type customDashboardWidgetConfigModel struct {
	ID   types.String `tfsdk:"id"`
	Size types.String `tfsdk:"size"`
}

type customDashboardExtraParametersModel struct {
	Description   types.String                       `tfsdk:"description"`
	WidgetsConfig []customDashboardWidgetConfigModel `tfsdk:"widgets_config"`
}

type customDashboardResourceModel struct {
	ID                types.String                         `tfsdk:"id"`
	Name              types.String                         `tfsdk:"name"`
	FilterData        map[string]interface{}               `tfsdk:"filter_data"`
	ExtraParameters   *customDashboardExtraParametersModel `tfsdk:"extra_params"`
	OrganizationLevel types.Bool                           `tfsdk:"organization_level"`
	ViewType          types.String                         `tfsdk:"view_type"`
}

func NewCustomDashboardResource() resource.Resource {
	return &customDashboardResource{}
}

func (r *customDashboardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_dashboard"
}

func (r *customDashboardResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *customDashboardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *customDashboardResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a custom dashboard resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Custom dashboard ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Custom dashboard title.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"filter_data": schema.MapAttribute{
				Description: "Should be left empty for custom dashboards.",
				ElementType: types.StringType,
				Required:    true,
			},
			"organization_level": schema.BoolAttribute{
				Description: "If set to true, it is a shared dashboard (can be viewed by any member of your Orca org). If set to false, it is a personal dashboard (can be viewed only by you, not other members of your Orca org).",
				Required:    true,
			},
			"view_type": schema.StringAttribute{
				Description: "Should be set to 'dashboard' for custom dashboards.",
				Required:    true,
			},
			"extra_params": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"description": schema.StringAttribute{
						Required: true,
					},
					"widgets_config": schema.ListNestedAttribute{
						Required: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Description: "ID of the identified widget.",
									Required:    true,
								},
								"size": schema.StringAttribute{
									Description: "Size of the identified widget. Possible values are sm (small), md (medium), or lg (large).",
									Required:    true,
								},
							},
						},
					},
				},
			},
		},
	}
}

// Widgets Config
func generateWidgetsConfig(plan *customDashboardExtraParametersModel) []api_client.WidgetConfig {
	var widgets_config []api_client.WidgetConfig

	for _, item := range plan.WidgetsConfig {
		widgets_config = append(widgets_config, api_client.WidgetConfig{
			ID:   item.ID.ValueString(),
			Size: item.Size.ValueString(),
		})
	}

	return widgets_config
}

// Extra Parameters
func generateExtraParameters(plan *customDashboardResourceModel) api_client.CustomDashboardExtraParameters {
	widgets_config := generateWidgetsConfig(plan.ExtraParameters)

	extra_params := api_client.CustomDashboardExtraParameters{
		Description:   plan.ExtraParameters.Description.ValueString(),
		WidgetsConfig: widgets_config,
	}

	return extra_params
}

func (r *customDashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customDashboardResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	filterData := make(map[string]interface{})

	createReq := api_client.CustomDashboard{
		Name:              plan.Name.ValueString(),
		FilterData:        filterData,
		ExtraParameters:   generateExtraParameters(&plan),
		OrganizationLevel: plan.OrganizationLevel.ValueBool(),
		ViewType:          plan.ViewType.ValueString(),
	}

	instance, err := r.apiClient.CreateCustomDashboard(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating custom dashboard",
			"Could not create custom dashboard, unexpected error: "+err.Error(),
		)
		return
	}

	instance, err = r.apiClient.GetCustomDashboard(instance.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error refreshing custom dashboard",
			"Could not create custom dashboard, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customDashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customDashboardResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.apiClient.DoesCustomDashboardExist(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading custom dashboard",
			fmt.Sprintf("Could not read custom dashboard ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	if !exists {
		tflog.Warn(ctx, fmt.Sprintf("Custom dashboard %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	instance, err := r.apiClient.GetCustomDashboard(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading custom dashboard",
			fmt.Sprintf("Could not read custom dashboard ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	state.ID = types.StringValue(instance.ID)
	state.Name = types.StringValue(instance.Name)
	state.OrganizationLevel = types.BoolValue(instance.OrganizationLevel)
	state.ViewType = types.StringValue(instance.ViewType)
	state.FilterData = make(map[string]interface{})

	var widget_settings []customDashboardWidgetConfigModel

	for _, item := range instance.ExtraParameters.WidgetsConfig {
		widget_settings = append(widget_settings, customDashboardWidgetConfigModel{
			ID:   types.StringValue(item.ID),
			Size: types.StringValue(item.Size),
		})
	}

	state.ExtraParameters = &customDashboardExtraParametersModel{
		Description:   types.StringValue(instance.ExtraParameters.Description),
		WidgetsConfig: widget_settings,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customDashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customDashboardResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.String() == "" {
		resp.Diagnostics.AddError(
			"ID is null",
			"Could not update custom dashboard, unexpected error: "+plan.ID.String(),
		)
		return
	}

	filterData := make(map[string]interface{})

	updateReq := api_client.CustomDashboard{
		ID:                plan.ID.ValueString(),
		Name:              plan.Name.ValueString(),
		FilterData:        filterData,
		ExtraParameters:   generateExtraParameters(&plan),
		OrganizationLevel: plan.OrganizationLevel.ValueBool(),
		ViewType:          plan.ViewType.ValueString(),
	}

	_, err := r.apiClient.UpdateCustomDashboard(updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating custom dashboard",
			"Could not create custom dashboard, unexpected error: "+err.Error(),
		)
		return
	}

	instance, err := r.apiClient.GetCustomDashboard(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating discovery view",
			"Could not read discovery view, unexpected error: "+err.Error(),
		)
		return
	}
	plan.ID = types.StringValue(instance.ID)
	plan.OrganizationLevel = types.BoolValue(instance.OrganizationLevel)
	plan.Name = types.StringValue(instance.Name)
	//plan.ExtraParameters = instance.ExtraParameters
	plan.ViewType = types.StringValue(instance.ViewType)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customDashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customDashboardResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteCustomDashboard(state.ID.String()[1 : len(state.ID.String())-1])
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting custom dashboard",
			"Could not delete custom dashboard, unexpected error: "+err.Error(),
		)
		return
	}
}
