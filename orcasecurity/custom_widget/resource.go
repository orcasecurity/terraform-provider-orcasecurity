package custom_widget

import (
	"context"
	"encoding/json"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

const errReadingCustomWidget = "Error reading custom widget"

type customWidgetResource struct {
	apiClient *api_client.APIClient
}

type customWidgetExtraParmetersSettingsFieldModel struct {
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

type customWidgetExtraParametersSettingsModel struct {
	Columns           types.List                                    `tfsdk:"columns"`
	Field             *customWidgetExtraParmetersSettingsFieldModel `tfsdk:"field"`
	RequestParameters requestParamsModel                            `tfsdk:"request_params"`
}

type requestParamsModel struct {
	Query            types.String   `tfsdk:"query"`
	GroupBy          []types.String `tfsdk:"group_by"`
	GroupByList      []types.String `tfsdk:"group_by_list"`
	Limit            types.Int64    `tfsdk:"limit"`
	OrderBy          types.List     `tfsdk:"order_by"`
	StartAtIndex     types.Int64    `tfsdk:"start_at_index"`
	EnablePagination types.Bool     `tfsdk:"enable_pagination"`
}

type customWidgetExtraParametersModel struct {
	Type              types.String                             `tfsdk:"type"`
	Category          types.String                             `tfsdk:"category"`
	EmptyStateMessage types.String                             `tfsdk:"empty_state_message"`
	Size              types.String                             `tfsdk:"default_size"`
	IsNew             types.Bool                               `tfsdk:"is_new"`
	Title             types.String                             `tfsdk:"title"`
	Subtitle          types.String                             `tfsdk:"subtitle"`
	Description       types.String                             `tfsdk:"description"`
	Settings          customWidgetExtraParametersSettingsModel `tfsdk:"settings"`
}

type customWidgetResourceModel struct {
	ID                types.String                      `tfsdk:"id"`
	Name              types.String                      `tfsdk:"name"`
	ExtraParameters   *customWidgetExtraParametersModel `tfsdk:"extra_params"`
	ViewType          types.String                      `tfsdk:"view_type"`
	OrganizationLevel types.Bool                        `tfsdk:"organization_level"`
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
		Description: "Provides a custom widget resource. According to Oxford Languages, a widget is an application, or a component of an interface, that enables a user to perform a function or access a service. Orca provides 50+ built-in widgets ([V1](https://docs.orcasecurity.io/v1/docs/available-dashboard-widgets) and [V2](https://docs.orcasecurity.io/docs/orca-dashboard-widgets-new)) that allow customers to more easily digest their cloud inventory and risks with certain filters. Customers can build custom widgets in cases where their use cases are more advanced than those covered by Orca's built-in widgets.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Custom widget ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "An internal, unique name for the widget.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"organization_level": schema.BoolAttribute{
				Description: "If set to true, it is a shared widget (can be viewed by any member of your Orca org). If set to false, it is a personal widget (can be viewed only by you, not other members of your Orca org).",
				Required:    true,
			},
			"view_type": schema.StringAttribute{
				Description: "This variable is `customs_widgets` for custom widgets.",
				Computed:    true,
			},
			"extra_params": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Type of custom widget to create. Valid values are `donut` and `table`. Legacy aliases `asset-table` and `alert-table` are also accepted; state will normalize to `table` for asset tables.",
						Required:    true,
					},
					"category": schema.StringAttribute{
						Description: "Should be set to 'custom' for custom dashboards.",
						Computed:    true,
					},
					"empty_state_message": schema.StringAttribute{
						Description: "When no objects are returned by the widget's underlying Discovery query, the widget would present this message.",
						Required:    true,
					},
					"default_size": schema.StringAttribute{
						Description: "Default size of the widget. Possible values are sm (small), md (medium), or lg (large).",
						Required:    true,
					},
					"is_new": schema.BoolAttribute{
						Description: "Should be set to true for a widget you are creating for the first time in Terraform.",
						Required:    true,
					},
					"title": schema.StringAttribute{
						Description: "Custom widget title that will be presented in the UI.",
						Computed:    true,
					},
					"subtitle": schema.StringAttribute{
						Description: "Custom widget subtitle that will be presented in the UI.",
						Required:    true,
					},
					"description": schema.StringAttribute{
						Description: "Custom widget description (the text that appears in the info bubble).",
						Required:    true,
					},
					"settings": schema.SingleNestedAttribute{
						Description: "These are the settings for the custom widget.",
						Required:    true,
						Attributes: map[string]schema.Attribute{
							"columns": schema.ListAttribute{
								Description: "Columns of the table. Required for table-type widgets. Not supported for donut-type widgets. First column to appear in the list will be the first column in the table widget; same thing for the next column in the list.",
								Optional:    true,
								ElementType: types.StringType,
							},
							"field": schema.SingleNestedAttribute{
								Description: "The name and type are also required here for grouping. This field is only required for donut-type widgets.",
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Description: "Name of the grouping method. For inventory-based queries, a common value is 'CloudAccount.Name'. To see other options, please use Chrome DevTools and the Orca UI to monitor what values this can be.",
										Required:    true,
									},
									"type": schema.StringAttribute{
										Description: "The name's type (normally 'str' for string).",
										Required:    true,
									},
								},
							},
							"request_params": schema.SingleNestedAttribute{
								Description: "These settings define the query and the grouping for the widget. For inventory-based queries, a common setting is to set 'group_by' to 'Type' and 'group_by_list' to 'CloudAccount.Name'.",
								Required:    true,
								Attributes: map[string]schema.Attribute{
									"query": schema.StringAttribute{
										Description: "Discovery query that the widget will use for its data.",
										Required:    true,
									},
									"group_by": schema.ListAttribute{
										ElementType: types.StringType,
										Description: "How to group the returned results.",
										Required:    true,
									},
									"group_by_list": schema.ListAttribute{
										ElementType: types.StringType,
										Description: "How to group the returned results. Do not use this option with the table-type widget",
										Optional:    true,
									},
									"limit": schema.Int64Attribute{
										Description: "Number of items returned in query.",
										Optional:    true,
									},
									"order_by": schema.ListAttribute{
										ElementType: types.StringType,
										Description: "How the returned items are ordered.",
										Optional:    true,
									},
									"start_at_index": schema.Int64Attribute{
										Optional: true,
									},
									"enable_pagination": schema.BoolAttribute{
										Optional: true,
									},
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
	if plan.Field != nil {
		return api_client.CustomWidgetExtraParametersSettingsField{
			Name: plan.Field.Name.ValueString(),
			Type: plan.Field.Type.ValueString(),
		}
	}
	// Return empty struct instead of nil
	return api_client.CustomWidgetExtraParametersSettingsField{}
}

func generateRequestParameters(plan *requestParamsModel) api_client.RequestParams {
	var request_params api_client.RequestParams

	queryString := plan.Query
	query := make(map[string]interface{})
	_ = json.Unmarshal([]byte(queryString.ValueString()), &query)

	group_by_string := make([]string, 0)
	group_by_list_string := make([]string, 0)

	for i := range plan.GroupBy {
		group_by_string = append(group_by_string, plan.GroupBy[i].ValueString())
	}
	if plan.GroupByList != nil {
		for j := range plan.GroupByList {
			group_by_list_string = append(group_by_list_string, plan.GroupByList[j].ValueString())
		}
	}

	// Orca API expects these additional models for discovery queries
	additionalModels := []string{"CloudAccount", "CustomTags", "BusinessUnits.Name"}

	var elements []string

	for _, v := range plan.OrderBy.Elements() {
		elements = append(elements, v.String()[1:len(v.String())-1])
	}

	request_params = api_client.RequestParams{
		Query:            query,
		GroupBy:          group_by_string,
		GroupByList:      group_by_list_string,
		AdditionalModels: additionalModels,
		Limit:            plan.Limit.ValueInt64(),
		OrderBy:          elements,
		StartAtIndex:     plan.StartAtIndex.ValueInt64(),
		EnablePagination: plan.EnablePagination.ValueBool(),
	}

	return request_params
}

func generateSettings(plan *customWidgetExtraParametersModel) []api_client.CustomWidgetExtraParametersSettings {
	var settings []api_client.CustomWidgetExtraParametersSettings
	var columns []string
	sizelist := [3]string{"sm", "md", "lg"}

	item := plan.Settings

	// Print each element as we process it
	for _, v := range plan.Settings.Columns.Elements() {
		columns = append(columns, (v.String())[1:len(v.String())-1])
	}

	for i := 0; i <= 2; i++ {
		field := generateField(&item)
		settings = append(settings, api_client.CustomWidgetExtraParametersSettings{
			Size:              sizelist[i],
			Columns:           columns,
			Field:             field,
			RequestParameters: generateRequestParameters(&item.RequestParameters),
		})
	}

	return settings
}

// Extra Parameters
func generateExtraParameters(plan *customWidgetResourceModel) api_client.CustomWidgetExtraParameters {

	settings := generateSettings(plan.ExtraParameters)

	var widgetType string

	if plan.ExtraParameters.Type.ValueString() == "donut" {
		widgetType = "PIE_CHART_SINGLE"
	} else if plan.ExtraParameters.Type.ValueString() == "asset-table" || plan.ExtraParameters.Type.ValueString() == "table" {
		widgetType = "ASSETS_TABLE"
	} else if plan.ExtraParameters.Type.ValueString() == "alert-table" {
		widgetType = "ALERTS_TABLE"
	} else {
		widgetType = plan.ExtraParameters.Type.ValueString()
	}

	requestParams := generateRequestParameters(&plan.ExtraParameters.Settings.RequestParameters)

	extra_params := api_client.CustomWidgetExtraParameters{
		Type:              widgetType,
		Category:          "Custom",
		EmptyStateMessage: plan.ExtraParameters.EmptyStateMessage.ValueString(),
		Size:              plan.ExtraParameters.Size.ValueString(),
		IsNew:             plan.ExtraParameters.IsNew.ValueBool(),
		Title:             plan.Name.ValueString(),
		Subtitle:          plan.ExtraParameters.Subtitle.ValueString(),
		Description:       plan.ExtraParameters.Description.ValueString(),
		RequestParams:     &requestParams,
		Settings:          settings,
	}

	return extra_params
}

// apiWidgetTypeToTerraform maps API widget type strings to Terraform schema values.
func apiWidgetTypeToTerraform(apiType string) string {
	switch apiType {
	case "PIE_CHART_SINGLE":
		return "donut"
	case "ASSETS_TABLE":
		// Canonical Terraform value is "table" (see schema docs).
		// "asset-table" is kept as a legacy alias on input only.
		return "table"
	case "ALERTS_TABLE":
		return "alert-table"
	default:
		return apiType
	}
}

// getRequestParams returns the effective request params. V2 API uses requestParams2;
// V1 uses requestParams. Prefer requestParams2 when present (V2-created widgets).
func getRequestParams(s api_client.CustomWidgetExtraParametersSettings) api_client.RequestParams {
	if s.RequestParams2 != nil {
		return *s.RequestParams2
	}
	return s.RequestParameters
}

// apiSettingsToStateSettings converts API settings to Terraform state model.
func apiSettingsToStateSettings(ctx context.Context, s api_client.CustomWidgetExtraParametersSettings) (customWidgetExtraParametersSettingsModel, error) {
	params := getRequestParams(s)
	queryJSON, err := json.Marshal(params.Query)
	if err != nil {
		return customWidgetExtraParametersSettingsModel{}, fmt.Errorf("marshaling request query: %w", err)
	}
	groupBy := stringSliceToTypesStrings(params.GroupBy)
	groupByList := stringSliceToTypesStrings(params.GroupByList)
	columns, err := columnsFromAPI(ctx, s.Columns)
	if err != nil {
		return customWidgetExtraParametersSettingsModel{}, fmt.Errorf("columns: %w", err)
	}
	orderBy, err := orderByFromAPI(ctx, params.OrderBy)
	if err != nil {
		return customWidgetExtraParametersSettingsModel{}, fmt.Errorf("order_by: %w", err)
	}
	settings := customWidgetExtraParametersSettingsModel{
		Columns: columns,
		RequestParameters: requestParamsModel{
			Query:            types.StringValue(string(queryJSON)),
			GroupBy:          groupBy,
			GroupByList:      groupByList,
			Limit:            types.Int64Value(params.Limit),
			StartAtIndex:     types.Int64Value(params.StartAtIndex),
			EnablePagination: types.BoolValue(params.EnablePagination),
			OrderBy:          orderBy,
		},
	}
	if s.Field.Name != "" || s.Field.Type != "" {
		settings.Field = &customWidgetExtraParmetersSettingsFieldModel{
			Name: types.StringValue(s.Field.Name),
			Type: types.StringValue(s.Field.Type),
		}
	}
	return settings, nil
}

func stringSliceToTypesStrings(ss []string) []types.String {
	out := make([]types.String, len(ss))
	for i, s := range ss {
		out[i] = types.StringValue(s)
	}
	return out
}

func columnsFromAPI(ctx context.Context, columns []string) (types.List, error) {
	if len(columns) == 0 {
		return types.ListNull(types.StringType), nil
	}
	out, diags := types.ListValueFrom(ctx, types.StringType, columns)
	if diags.HasError() {
		return types.ListNull(types.StringType), diagError(diags)
	}
	return out, nil
}

func orderByFromAPI(ctx context.Context, orderBy []string) (types.List, error) {
	if len(orderBy) == 0 {
		return types.ListNull(types.StringType), nil
	}
	out, diags := types.ListValueFrom(ctx, types.StringType, stringSliceToTypesStrings(orderBy))
	if diags.HasError() {
		return types.ListNull(types.StringType), diagError(diags)
	}
	return out, nil
}

// diagError returns a single error from the first diagnostic (for propagation to Read diagnostics).
func diagError(diags diag.Diagnostics) error {
	if !diags.HasError() {
		return nil
	}
	e := diags.Errors()[0]
	return fmt.Errorf("%s: %s", e.Summary(), e.Detail())
}

// instanceToState maps API CustomWidget to Terraform state model. Used by Read (including import).
func instanceToState(ctx context.Context, instance *api_client.CustomWidget) (customWidgetResourceModel, error) {
	ep := instance.ExtraParameters
	settings := customWidgetExtraParametersSettingsModel{}
	if len(ep.Settings) > 0 {
		var err error
		settings, err = apiSettingsToStateSettings(ctx, ep.Settings[0])
		if err != nil {
			return customWidgetResourceModel{}, err
		}
	}

	return customWidgetResourceModel{
		ID:                types.StringValue(instance.ID),
		Name:              types.StringValue(instance.Name),
		OrganizationLevel: types.BoolValue(instance.OrganizationLevel),
		ViewType:          types.StringValue(instance.ViewType),
		ExtraParameters: &customWidgetExtraParametersModel{
			Type:              types.StringValue(apiWidgetTypeToTerraform(ep.Type)),
			Category:          types.StringValue(ep.Category),
			EmptyStateMessage: types.StringValue(ep.EmptyStateMessage),
			Size:              types.StringValue(ep.Size),
			IsNew:             types.BoolValue(ep.IsNew),
			Title:             types.StringValue(ep.Title),
			Subtitle:          types.StringValue(ep.Subtitle),
			Description:       types.StringValue(ep.Description),
			Settings:          settings,
		},
	}, nil
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
		ViewType:          "customs_widgets",
	}

	instance, err := r.apiClient.CreateCustomWidget(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating widget",
			"Could not create widget, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.ViewType = types.StringValue("customs_widgets")
	plan.ExtraParameters.Category = types.StringValue("Custom")
	plan.ExtraParameters.Title = plan.Name

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customWidgetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Starting Read operation")

	var state customWidgetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	exists, err := r.apiClient.DoesCustomWidgetExist(id)
	if err != nil {
		resp.Diagnostics.AddError(
			errReadingCustomWidget,
			fmt.Sprintf("Could not read custom widget ID %s: %s", id, err.Error()),
		)
		return
	}

	if !exists {
		tflog.Warn(ctx, fmt.Sprintf("Custom widget %s is missing on the remote side.", id))
		resp.State.RemoveResource(ctx)
		return
	}

	instance, err := r.apiClient.GetCustomWidget(id)
	if err != nil {
		resp.Diagnostics.AddError(
			errReadingCustomWidget,
			fmt.Sprintf("Could not read custom widget ID %s: %s", id, err.Error()),
		)
		return
	}

	state, err = instanceToState(ctx, instance)
	if err != nil {
		resp.Diagnostics.AddError(
			errReadingCustomWidget,
			fmt.Sprintf("Could not convert widget state for ID %s: %s", id, err.Error()),
		)
		return
	}
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *customWidgetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	tflog.Info(ctx, "Starting Update operation")

	var plan customWidgetResourceModel
	diags := req.Plan.Get(ctx, &plan)

	tflog.Info(ctx, fmt.Sprintf("Plan ID before update: %s", plan.ID.ValueString()))
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

	api_client_extra_parameters := generateExtraParameters(&plan)

	updateReq := api_client.CustomWidget{
		ID:                plan.ID.ValueString(),
		Name:              plan.Name.ValueString(),
		FilterData:        make(map[string]interface{}),
		ExtraParameters:   api_client_extra_parameters,
		OrganizationLevel: plan.OrganizationLevel.ValueBool(),
		ViewType:          "customs_widgets",
	}

	instance, err := r.apiClient.UpdateCustomWidget(updateReq)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating custom widget",
			"Could not update custom widget, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Received instance ID after update: %s", instance.ID))

	plan.ID = types.StringValue(instance.ID)
	tflog.Info(ctx, fmt.Sprintf("Plan ID being set in state: %s", plan.ID.ValueString()))
	plan.OrganizationLevel = types.BoolValue(instance.OrganizationLevel)
	plan.Name = types.StringValue(instance.Name)
	plan.ViewType = types.StringValue(instance.ViewType)
	plan.ExtraParameters.Category = types.StringValue("Custom")
	plan.ExtraParameters.Title = types.StringValue(instance.ExtraParameters.Title)

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
