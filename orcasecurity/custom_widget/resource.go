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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
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

const (
	errReadingCustomWidget = "Error reading custom widget"
	tfTypeAlertTable       = "alert-table"
)

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
	RequestParameters *requestParamsModel                           `tfsdk:"request_params"`
	RequestParamsList []comparisonRequestParamModel                 `tfsdk:"request_params_list"`
}

type comparisonRequestParamModel struct {
	ID      types.String   `tfsdk:"id"`
	Title   types.String   `tfsdk:"title"`
	Query   types.String   `tfsdk:"query"`
	GroupBy []types.String `tfsdk:"group_by"`
}

type widgetInnerExtraParamsModel struct {
	Field         *customWidgetExtraParmetersSettingsFieldModel `tfsdk:"field"`
	ValuesFormat  types.String                                  `tfsdk:"values_format"`
	DefaultMapper types.String                                  `tfsdk:"default_mapper"`
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
	WidgetExtraParams *widgetInnerExtraParamsModel             `tfsdk:"widget_extra_params"`
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"extra_params": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Type of custom widget to create. Valid values: `donut` (PIE_CHART_SINGLE), `table` (ASSETS_TABLE), `alert-table` (ALERTS_TABLE), `metric` (ICON_GRID), `comparison` (PIE_CHART_MULTI). Legacy alias `asset-table` accepted; state normalizes to `table`.",
						Required:    true,
					},
					"category": schema.StringAttribute{
						Description: "Should be set to 'custom' for custom dashboards.",
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"empty_state_message": schema.StringAttribute{
						Description: "When no objects are returned by the widget's underlying Discovery query, the widget would present this message.",
						Required:    true,
					},
					"default_size": schema.StringAttribute{
						Description: "Default size of the widget. Values: sm (1/3 width), md (2/3 width), lg (3/4 width in V2, full in V1), xl (full width, V2 only). See custom_dashboard docs for Widget Sizes.",
						Required:    true,
					},
					"is_new": schema.BoolAttribute{
						Description: "Should be set to true for a widget you are creating for the first time in Terraform.",
						Required:    true,
					},
					"title": schema.StringAttribute{
						Description: "Custom widget title that will be presented in the UI.",
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"subtitle": schema.StringAttribute{
						Description: "Custom widget subtitle that will be presented in the UI.",
						Required:    true,
					},
					"description": schema.StringAttribute{
						Description: "Custom widget description (the text that appears in the info bubble).",
						Required:    true,
					},
					"widget_extra_params": schema.SingleNestedAttribute{
						Description: "Extra params block (`extraParams` in API). Used by `comparison` (PIE_CHART_MULTI) widgets to hold field, default_mapper, and values_format.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"field": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{Required: true},
									"type": schema.StringAttribute{Required: true},
								},
							},
							"values_format": schema.StringAttribute{
								Description: "UI values format (e.g. `default`).",
								Optional:    true,
							},
							"default_mapper": schema.StringAttribute{
								Description: "JSON-encoded default mapper (object). Example: jsonencode({ main = { color = \"...\" }, comparison = { color = \"...\" } }).",
								Optional:    true,
							},
						},
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
								Description: "Query and grouping for the widget. Required for donut/table/alert-table/metric. Omit for `comparison` widgets (use `request_params_list` instead).",
								Optional:    true,
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
										Computed:    true,
										PlanModifiers: []planmodifier.List{
											listplanmodifier.UseStateForUnknown(),
										},
									},
									"limit": schema.Int64Attribute{
										Description: "Number of items returned in query.",
										Optional:    true,
										Computed:    true,
										PlanModifiers: []planmodifier.Int64{
											int64planmodifier.UseStateForUnknown(),
										},
									},
									"order_by": schema.ListAttribute{
										ElementType: types.StringType,
										Description: "How the returned items are ordered.",
										Optional:    true,
										Computed:    true,
										PlanModifiers: []planmodifier.List{
											listplanmodifier.UseStateForUnknown(),
										},
									},
									"start_at_index": schema.Int64Attribute{
										Optional: true,
										Computed: true,
										PlanModifiers: []planmodifier.Int64{
											int64planmodifier.UseStateForUnknown(),
										},
									},
									"enable_pagination": schema.BoolAttribute{
										Optional: true,
										Computed: true,
										PlanModifiers: []planmodifier.Bool{
											boolplanmodifier.UseStateForUnknown(),
										},
									},
								},
							},
							"request_params_list": schema.ListNestedAttribute{
								Description: "List of named query param sets for `comparison` (PIE_CHART_MULTI) widgets. Typically two entries: `main` and `comparison`.",
								Optional:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"id":    schema.StringAttribute{Required: true},
										"title": schema.StringAttribute{Required: true},
										"query": schema.StringAttribute{Required: true},
										"group_by": schema.ListAttribute{
											ElementType: types.StringType,
											Required:    true,
										},
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

func generateField(plan *customWidgetExtraParametersSettingsModel) *api_client.CustomWidgetExtraParametersSettingsField {
	if plan.Field == nil {
		return nil
	}
	return &api_client.CustomWidgetExtraParametersSettingsField{
		Name: plan.Field.Name.ValueString(),
		Type: plan.Field.Type.ValueString(),
	}
}

// additionalModelsFor returns the default additional_models[] for a given Terraform
// widget type. Donut/metric/comparison widgets include CustomTags + BusinessUnits;
// table/alert-table widgets include only CloudAccount.
func additionalModelsFor(tfType string) []string {
	switch tfType {
	case "donut", "metric", "comparison":
		return []string{"CloudAccount", "CustomTags", "BusinessUnits.Name"}
	default:
		return []string{"CloudAccount"}
	}
}

// needsGroupByBracket reports whether this widget type needs the legacy
// `group_by[]` key duplicated alongside `group_by` (server requires both for
// PIE_CHART_SINGLE / ICON_GRID / PIE_CHART_MULTI).
func needsGroupByBracket(tfType string) bool {
	switch tfType {
	case "donut", "metric", "comparison":
		return true
	default:
		return false
	}
}

// supportsPagination reports whether this widget type uses start_at_index/enable_pagination
// (table/alert-table only). Server 500s when those keys are present for metric/donut/comparison.
func supportsPagination(tfType string) bool {
	switch tfType {
	case "table", "asset-table", tfTypeAlertTable:
		return true
	default:
		return false
	}
}

func generateRequestParameters(plan *requestParamsModel, additionalModels []string, withBracket bool, withPagination bool) api_client.RequestParams {
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

	var elements []string
	for _, v := range plan.OrderBy.Elements() {
		elements = append(elements, v.String()[1:len(v.String())-1])
	}

	rp := api_client.RequestParams{
		Query:            query,
		GroupBy:          group_by_string,
		GroupByList:      group_by_list_string,
		AdditionalModels: additionalModels,
		Limit:            plan.Limit.ValueInt64(),
		OrderBy:          elements,
	}
	if withBracket {
		rp.GroupByBracket = group_by_string
	}
	if withPagination {
		sai := plan.StartAtIndex.ValueInt64()
		ep := plan.EnablePagination.ValueBool()
		rp.StartAtIndex = &sai
		rp.EnablePagination = &ep
	}
	return rp
}

func generateComparisonParams(list []comparisonRequestParamModel, additionalModels []string) []api_client.ComparisonRequestParam {
	out := make([]api_client.ComparisonRequestParam, 0, len(list))
	for _, p := range list {
		query := make(map[string]interface{})
		_ = json.Unmarshal([]byte(p.Query.ValueString()), &query)
		groupBy := make([]string, 0, len(p.GroupBy))
		for _, g := range p.GroupBy {
			groupBy = append(groupBy, g.ValueString())
		}
		out = append(out, api_client.ComparisonRequestParam{
			ID:    p.ID.ValueString(),
			Title: p.Title.ValueString(),
			Params: api_client.RequestParams{
				Query:            query,
				GroupBy:          groupBy,
				GroupByBracket:   groupBy,
				AdditionalModels: additionalModels,
			},
		})
	}
	return out
}

func generateWidgetExtraParams(plan *widgetInnerExtraParamsModel) *api_client.WidgetInnerExtraParams {
	if plan == nil {
		return nil
	}
	out := &api_client.WidgetInnerExtraParams{
		ValuesFormat: plan.ValuesFormat.ValueString(),
	}
	if plan.Field != nil {
		out.Field = &api_client.CustomWidgetExtraParametersSettingsField{
			Name: plan.Field.Name.ValueString(),
			Type: plan.Field.Type.ValueString(),
		}
	}
	if !plan.DefaultMapper.IsNull() && plan.DefaultMapper.ValueString() != "" {
		m := make(map[string]interface{})
		_ = json.Unmarshal([]byte(plan.DefaultMapper.ValueString()), &m)
		out.DefaultMapper = m
	}
	return out
}

// tfTypeToAPI maps Terraform widget type strings to API widget type strings.
func tfTypeToAPI(tfType string) string {
	switch tfType {
	case "donut":
		return "PIE_CHART_SINGLE"
	case "table", "asset-table":
		return "ASSETS_TABLE"
	case tfTypeAlertTable:
		return "ALERTS_TABLE"
	case "metric":
		return "ICON_GRID"
	case "comparison":
		return "PIE_CHART_MULTI"
	default:
		return tfType
	}
}

// apiWidgetTypeToTerraform maps API widget type strings to Terraform schema values.
func apiWidgetTypeToTerraform(apiType string) string {
	switch apiType {
	case "PIE_CHART_SINGLE":
		return "donut"
	case "ASSETS_TABLE":
		return "table"
	case "ALERTS_TABLE":
		return tfTypeAlertTable
	case "ICON_GRID":
		return "metric"
	case "PIE_CHART_MULTI":
		return "comparison"
	default:
		return apiType
	}
}

func generateSettings(plan *customWidgetExtraParametersModel) []api_client.CustomWidgetExtraParametersSettings {
	item := plan.Settings
	tfType := plan.Type.ValueString()
	additionalModels := additionalModelsFor(tfType)

	var columns []string
	for _, v := range plan.Settings.Columns.Elements() {
		columns = append(columns, (v.String())[1:len(v.String())-1])
	}

	field := generateField(&item)
	innerExtra := generateWidgetExtraParams(plan.WidgetExtraParams)

	// Marshal requestParams2 polymorphically: array for comparison, object otherwise.
	var rp2Bytes []byte
	if tfType == "comparison" && len(item.RequestParamsList) > 0 {
		list := generateComparisonParams(item.RequestParamsList, additionalModels)
		rp2Bytes, _ = json.Marshal(list)
	} else if item.RequestParameters != nil {
		single := generateRequestParameters(item.RequestParameters, additionalModels, needsGroupByBracket(tfType), supportsPagination(tfType))
		rp2Bytes, _ = json.Marshal(single)
	}

	// Dashboard renderer looks up settings entry by size. Emit one entry per size
	// (sm/md/lg) so widget renders regardless of dashboard cell size.
	sizes := []string{"sm", "md", "lg"}
	settings := make([]api_client.CustomWidgetExtraParametersSettings, 0, len(sizes))
	for _, s := range sizes {
		entry := api_client.CustomWidgetExtraParametersSettings{
			Size:           s,
			Columns:        columns,
			Field:          field,
			ExtraParams:    innerExtra,
			RequestParams2: rp2Bytes,
		}
		settings = append(settings, entry)
	}

	return settings
}

// Extra Parameters
func generateExtraParameters(plan *customWidgetResourceModel) api_client.CustomWidgetExtraParameters {
	settings := generateSettings(plan.ExtraParameters)
	tfType := plan.ExtraParameters.Type.ValueString()
	widgetType := tfTypeToAPI(tfType)

	extra_params := api_client.CustomWidgetExtraParameters{
		Type:              widgetType,
		Category:          "Custom",
		EmptyStateMessage: plan.ExtraParameters.EmptyStateMessage.ValueString(),
		Size:              plan.ExtraParameters.Size.ValueString(),
		IsNew:             plan.ExtraParameters.IsNew.ValueBool(),
		Title:             plan.Name.ValueString(),
		Subtitle:          plan.ExtraParameters.Subtitle.ValueString(),
		Description:       plan.ExtraParameters.Description.ValueString(),
		ExtraParams:       generateWidgetExtraParams(plan.ExtraParameters.WidgetExtraParams),
		Settings:          settings,
	}

	// PIE_CHART_MULTI needs the comparison params array at the top level too.
	if tfType == "comparison" && len(plan.ExtraParameters.Settings.RequestParamsList) > 0 {
		list := generateComparisonParams(plan.ExtraParameters.Settings.RequestParamsList, additionalModelsFor(tfType))
		if b, err := json.Marshal(list); err == nil {
			extra_params.RequestParams = b
		}
	}

	return extra_params
}

// getRequestParams returns the effective single request params. V2 API uses
// requestParams2 (RawMessage), V1 uses requestParams. For comparison widgets
// where requestParams2 is an array, returns the first entry's params.
func getRequestParams(s api_client.CustomWidgetExtraParametersSettings) api_client.RequestParams {
	if len(s.RequestParams2) > 0 {
		trimmed := bytesTrimSpace(s.RequestParams2)
		if len(trimmed) > 0 && trimmed[0] == '[' {
			var list []api_client.ComparisonRequestParam
			if err := json.Unmarshal(s.RequestParams2, &list); err == nil && len(list) > 0 {
				return list[0].Params
			}
		} else {
			var single api_client.RequestParams
			if err := json.Unmarshal(s.RequestParams2, &single); err == nil {
				return single
			}
		}
	}
	if s.RequestParameters != nil {
		return *s.RequestParameters
	}
	return api_client.RequestParams{}
}

// getComparisonParams returns the parsed comparison param list if requestParams2 is an array.
func getComparisonParams(s api_client.CustomWidgetExtraParametersSettings) []api_client.ComparisonRequestParam {
	if len(s.RequestParams2) == 0 {
		return nil
	}
	trimmed := bytesTrimSpace(s.RequestParams2)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil
	}
	var list []api_client.ComparisonRequestParam
	if err := json.Unmarshal(s.RequestParams2, &list); err != nil {
		return nil
	}
	return list
}

func bytesTrimSpace(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\t' || b[0] == '\n' || b[0] == '\r') {
		b = b[1:]
	}
	return b
}

// apiSettingsToStateSettings converts API settings to Terraform state model.
func apiSettingsToStateSettings(ctx context.Context, s api_client.CustomWidgetExtraParametersSettings) (customWidgetExtraParametersSettingsModel, error) {
	columns, err := columnsFromAPI(ctx, s.Columns)
	if err != nil {
		return customWidgetExtraParametersSettingsModel{}, fmt.Errorf("columns: %w", err)
	}
	settings := customWidgetExtraParametersSettingsModel{Columns: columns}

	if comparison := getComparisonParams(s); comparison != nil {
		settings.RequestParamsList = comparisonParamsToState(comparison)
	} else {
		rp, rpErr := singleParamsToState(ctx, s)
		if rpErr != nil {
			return customWidgetExtraParametersSettingsModel{}, rpErr
		}
		settings.RequestParameters = rp
	}

	if s.Field != nil && (s.Field.Name != "" || s.Field.Type != "") {
		settings.Field = &customWidgetExtraParmetersSettingsFieldModel{
			Name: types.StringValue(s.Field.Name),
			Type: types.StringValue(s.Field.Type),
		}
	}
	return settings, nil
}

func comparisonParamsToState(comparison []api_client.ComparisonRequestParam) []comparisonRequestParamModel {
	list := make([]comparisonRequestParamModel, 0, len(comparison))
	for _, c := range comparison {
		qJSON, _ := json.Marshal(c.Params.Query)
		list = append(list, comparisonRequestParamModel{
			ID:      types.StringValue(c.ID),
			Title:   types.StringValue(c.Title),
			Query:   types.StringValue(string(qJSON)),
			GroupBy: stringSliceToTypesStrings(c.Params.GroupBy),
		})
	}
	return list
}

func singleParamsToState(ctx context.Context, s api_client.CustomWidgetExtraParametersSettings) (*requestParamsModel, error) {
	params := getRequestParams(s)
	queryJSON, mErr := json.Marshal(params.Query)
	if mErr != nil {
		return nil, fmt.Errorf("marshaling request query: %w", mErr)
	}
	orderBy, oErr := orderByFromAPI(ctx, params.OrderBy)
	if oErr != nil {
		return nil, fmt.Errorf("order_by: %w", oErr)
	}
	var sai int64
	if params.StartAtIndex != nil {
		sai = *params.StartAtIndex
	}
	var ep bool
	if params.EnablePagination != nil {
		ep = *params.EnablePagination
	}
	return &requestParamsModel{
		Query:            types.StringValue(string(queryJSON)),
		GroupBy:          stringSliceToTypesStrings(params.GroupBy),
		GroupByList:      stringSliceToTypesStrings(params.GroupByList),
		Limit:            types.Int64Value(params.Limit),
		StartAtIndex:     types.Int64Value(sai),
		EnablePagination: types.BoolValue(ep),
		OrderBy:          orderBy,
	}, nil
}

func widgetExtraParamsToState(ep *api_client.WidgetInnerExtraParams) *widgetInnerExtraParamsModel {
	if ep == nil {
		return nil
	}
	out := &widgetInnerExtraParamsModel{
		ValuesFormat:  types.StringValue(ep.ValuesFormat),
		DefaultMapper: types.StringNull(),
	}
	if ep.Field != nil {
		out.Field = &customWidgetExtraParmetersSettingsFieldModel{
			Name: types.StringValue(ep.Field.Name),
			Type: types.StringValue(ep.Field.Type),
		}
	}
	if ep.DefaultMapper != nil {
		if b, err := json.Marshal(ep.DefaultMapper); err == nil {
			out.DefaultMapper = types.StringValue(string(b))
		}
	}
	return out
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
	out, diags := types.ListValueFrom(ctx, types.StringType, orderBy)
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
			WidgetExtraParams: widgetExtraParamsToState(ep.ExtraParams),
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
