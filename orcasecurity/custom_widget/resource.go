package custom_widget

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
	_ resource.Resource                = &customWidgetResource{}
	_ resource.ResourceWithConfigure   = &customWidgetResource{}
	_ resource.ResourceWithImportState = &customWidgetResource{}
)

type customWidgetResource struct {
	apiClient *api_client.APIClient
}

type customWidgetExtraParmetersSettingsFieldModel struct {
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

type customWidgetExtraParametersSettingsModel struct {
	Size              types.String                                  `tfsdk:"size"`
	Field             *customWidgetExtraParmetersSettingsFieldModel `tfsdk:"field"`
	RequestParameters types.String                                  `tfsdk:"request_params"`
}

type customWidgetExtraParametersModel struct {
	Type              types.String                               `tfsdk:"type"`
	Category          types.String                               `tfsdk:"category"`
	EmptyStateMessage types.String                               `tfsdk:"empty_state_message"`
	Size              types.String                               `tfsdk:"size"`
	IsNew             types.Bool                                 `tfsdk:"is_new"`
	Title             types.String                               `tfsdk:"title"`
	Subtitle          types.String                               `tfsdk:"subtitle"`
	Description       types.String                               `tfsdk:"description"`
	Settings          []customWidgetExtraParametersSettingsModel `tfsdk:"settings"`
}

type customWidgetResourceModel struct {
	ID                types.String                      `tfsdk:"id"`
	Name              types.String                      `tfsdk:"name"`
	FilterData        map[string]interface{}            `tfsdk:"filter_data"`
	ExtraParameters   *customWidgetExtraParametersModel `tfsdk:"extra_params"`
	OrganizationLevel types.Bool                        `tfsdk:"organization_level"`
	ViewType          types.String                      `tfsdk:"view_type"`
}

func NewCustomWidgetResource() resource.Resource {
	return &customWidgetResource{}
}

func (r *customWidgetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_widget"
}

func (r *customWidgetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *customWidgetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *customWidgetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	//tflog.Error(ctx, "Setting up Schema")
	resp.Schema = schema.Schema{
		Description: "Provides a custom widget resource. According to Oxford Languages, a widget is an application, or a component of an interface, that enables a user to perform a function or access a service. Orca provides 50+ built-in widgets (https://docs.orcasecurity.io/v1/docs/available-dashboard-widgets) that allow customers to more easily digest their cloud inventory and risks with certain filters. Customers can build custom widgets in cases where their use cases are more advanced than those covered by Orca's built-in widgets.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Custom widget ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Custom widget title.",
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
				Description: "If set to true, it is a shared widget (can be viewed by any member of your Orca org). If set to false, it is a personal widget (can be viewed only by you, not other members of your Orca org).",
				Required:    true,
			},
			"view_type": schema.StringAttribute{
				Description: "Should be set to 'customs_widgets' for custom dashboards.",
				Required:    true,
			},
			"extra_params": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Type of custom widget to create. Can be set to 'PIE_CHART_SINGLE' (for pie charts) only, at the moment.",
						Required:    true,
					},
					"category": schema.StringAttribute{
						Description: "Should be set to 'custom' for custom dashboards.",
						Required:    true,
					},
					"empty_state_message": schema.StringAttribute{
						Description: "When no objects are returned by the widget's underlying Discovery query, the widget would present this message.",
						Required:    true,
					},
					"size": schema.StringAttribute{
						Description: "Defautl size of the identified widget. Possible values are sm (small), md (medium), or lg (large).",
						Required:    true,
					},
					"is_new": schema.BoolAttribute{
						Description: "Should be set to true for a widget you are creating for the first time in Terraform.",
						Required:    true,
					},
					"title": schema.StringAttribute{
						Description: "Custom widget title.",
						Required:    true,
					},
					"subtitle": schema.StringAttribute{
						Description: "Custom widget subtitle.",
						Required:    true,
					},
					"description": schema.StringAttribute{
						Description: "Custom widget description (the text that appears in the info bubble).",
						Required:    true,
					},
					"settings": schema.ListNestedAttribute{
						Required: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"size": schema.StringAttribute{
									Description: "Size of the custom widget. Possible values are sm (small), md (medium), or lg (large).",
									Required:    true,
								},
								"field": schema.SingleNestedAttribute{
									Required: true,
									Attributes: map[string]schema.Attribute{
										"name": schema.StringAttribute{
											Required: true,
										},
										"type": schema.StringAttribute{
											Required: true,
										},
									},
								},
								"request_params": schema.StringAttribute{
									Required: true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func generateField(plan *customWidgetExtraParametersSettingsModel) api_client.CustomWidgetExtraParametersSettingsField {
	field := api_client.CustomWidgetExtraParametersSettingsField{
		Name: plan.Field.Name.ValueString(),
		Type: plan.Field.Type.ValueString(),
	}

	return field
}

// Settings
func generateSettings(plan *customWidgetExtraParametersModel) []api_client.CustomWidgetExtraParametersSettings {
	var settings []api_client.CustomWidgetExtraParametersSettings

	for _, item := range plan.Settings {
		settings = append(settings, api_client.CustomWidgetExtraParametersSettings{
			Size:              item.Size.ValueString(),
			Field:             generateField(&item),
			RequestParameters: item.RequestParameters.ValueString(),
		})
	}

	return settings
}

// Extra Parameters
func generateExtraParameters(plan *customWidgetResourceModel) api_client.CustomWidgetExtraParameters {

	settings := generateSettings(plan.ExtraParameters)

	extra_params := api_client.CustomWidgetExtraParameters{
		Type:              plan.ExtraParameters.Type.ValueString(),
		Category:          plan.ExtraParameters.Category.ValueString(),
		EmptyStateMessage: plan.ExtraParameters.EmptyStateMessage.ValueString(),
		Size:              plan.ExtraParameters.Size.ValueString(),
		IsNew:             plan.ExtraParameters.IsNew.ValueBool(),
		Title:             plan.ExtraParameters.Title.ValueString(),
		Subtitle:          plan.ExtraParameters.Subtitle.ValueString(),
		Description:       plan.ExtraParameters.Description.ValueString(),
		Settings:          settings,
	}

	return extra_params
}

func (r *customWidgetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customWidgetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	filterData := make(map[string]interface{})

	createReq := api_client.CustomWidget{
		Name:              plan.Name.ValueString(),
		FilterData:        filterData,
		ExtraParameters:   generateExtraParameters(&plan),
		OrganizationLevel: plan.OrganizationLevel.ValueBool(),
		ViewType:          plan.ViewType.ValueString(),
	}

	instance, err := r.apiClient.CreateCustomWidget(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating widget",
			"Could not create widget, unexpected error: "+err.Error(),
		)
		return
	}

	instance, err = r.apiClient.GetCustomWidget(instance.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error refreshing custom widget with error",
			"Could not refresh custom widget, unexpected error and Instace ID: .",
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

func (r *customWidgetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customWidgetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.apiClient.DoesCustomWidgetExist(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading custom widget",
			fmt.Sprintf("Could not read custom widget ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	if !exists {
		tflog.Warn(ctx, fmt.Sprintf("Custom widget %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	instance, err := r.apiClient.GetCustomWidget(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading custom widget",
			fmt.Sprintf("Could not read custom widget ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	state.ID = types.StringValue(instance.ID)
	state.Name = types.StringValue(instance.Name)
	state.OrganizationLevel = types.BoolValue(instance.OrganizationLevel)
	state.ViewType = types.StringValue(instance.ViewType)
	state.FilterData = instance.FilterData

	var settings []customWidgetExtraParametersSettingsModel

	for _, item := range instance.ExtraParameters.Settings {
		settings = append(state.ExtraParameters.Settings, customWidgetExtraParametersSettingsModel{
			Size:              types.StringValue(item.Size),
			Field:             &customWidgetExtraParmetersSettingsFieldModel{Name: types.StringValue(item.Field.Name), Type: types.StringValue(item.Field.Type)},
			RequestParameters: types.StringValue(item.RequestParameters),
		})
	}

	state.ExtraParameters = &customWidgetExtraParametersModel{
		Type:              types.StringValue(instance.ExtraParameters.Type),
		Category:          types.StringValue(instance.ExtraParameters.Category),
		EmptyStateMessage: types.StringValue(instance.ExtraParameters.EmptyStateMessage),
		Size:              types.StringValue(instance.ExtraParameters.Size),
		IsNew:             types.BoolValue(instance.ExtraParameters.IsNew),
		Title:             types.StringValue(instance.ExtraParameters.Title),
		Subtitle:          types.StringValue(instance.ExtraParameters.Subtitle),
		Description:       types.StringValue(instance.ExtraParameters.Description),
		Settings:          settings,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customWidgetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customWidgetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.String() == "" {
		resp.Diagnostics.AddError(
			"ID is null",
			"Could not update custom widget, unexpected error: "+plan.ID.ValueString(),
		)
		return
	}

	var settings []api_client.CustomWidgetExtraParametersSettings

	for _, item := range plan.ExtraParameters.Settings {
		settings = append(settings, api_client.CustomWidgetExtraParametersSettings{
			Size:              item.Size.ValueString(),
			Field:             api_client.CustomWidgetExtraParametersSettingsField{Name: item.Field.Name.ValueString(), Type: item.Field.Type.ValueString()},
			RequestParameters: item.RequestParameters.ValueString(),
		})
	}

	api_client_extra_parameters := api_client.CustomWidgetExtraParameters{
		Type:              plan.ExtraParameters.Type.ValueString(),
		Category:          plan.ExtraParameters.Category.ValueString(),
		EmptyStateMessage: plan.ExtraParameters.EmptyStateMessage.ValueString(),
		Size:              plan.ExtraParameters.Size.ValueString(),
		IsNew:             plan.ExtraParameters.IsNew.ValueBool(),
		Title:             plan.ExtraParameters.Title.ValueString(),
		Subtitle:          plan.ExtraParameters.Subtitle.ValueString(),
		Description:       plan.ExtraParameters.Description.ValueString(),
		Settings:          settings,
	}

	updateReq := api_client.CustomWidget{
		ID:                plan.ID.ValueString(),
		Name:              plan.Name.ValueString(),
		FilterData:        plan.FilterData,
		ExtraParameters:   api_client_extra_parameters,
		OrganizationLevel: plan.OrganizationLevel.ValueBool(),
		ViewType:          plan.ViewType.ValueString(),
	}

	instance, err := r.apiClient.UpdateCustomWidget(updateReq)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating custom widget",
			"Could not create custom widget, unexpected error: "+err.Error(),
		)
		return
	}

	instance, err = r.apiClient.GetCustomWidget(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating discovery view",
			"Could not read discovery view, unexpected error: "+err.Error(),
		)
		return
	}

	/*
		plan.ExtraParameters = &customWidgetExtraParametersModel{
			Type:              types.StringValue(instance.ExtraParameters.Type),
			Category:          types.StringValue(instance.ExtraParameters.Category),
			EmptyStateMessage: types.StringValue(instance.ExtraParameters.EmptyStateMessage),
			Size:              types.StringValue(instance.ExtraParameters.Size),
			IsNew:             types.BoolValue(instance.ExtraParameters.IsNew),
			Title:             types.StringValue(instance.ExtraParameters.Title),
			Subtitle:          types.StringValue(instance.ExtraParameters.Subtitle),
			Description:       types.StringValue(instance.ExtraParameters.Description),
			Settings:          settings,
		}*/

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

func (r *customWidgetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customWidgetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteCustomWidget(state.ID.String()[1 : len(state.ID.String())-1])
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting custom widget",
			"Could not delete custom widget, unexpected error: "+err.Error(),
		)
		return
	}
}
