package discovery_view

import (
	"context"
	"encoding/json"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
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

const errReadingDiscoveryView = "Error reading discovery view"

type discoveryViewResource struct {
	apiClient *api_client.APIClient
}

type personalViewWarningValidator struct{}

func (personalViewWarningValidator) Description(_ context.Context) string {
	return "warns when organization_level is set to false"
}

func (v personalViewWarningValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (personalViewWarningValidator) ValidateBool(_ context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if !req.ConfigValue.ValueBool() {
		resp.Diagnostics.AddAttributeWarning(
			req.Path,
			"Personal discovery view is scoped to the API token user",
			"organization_level = false creates a personal view owned by the user identity behind the provider's API token. "+
				"It will only appear in the Orca UI when you log in as that same user. If the token user differs from your UI user, "+
				"the view will exist in the API but will not be visible in the UI. Set organization_level = true to share with the org.",
		)
	}
}

type discoveryQueryResourceModel struct {
	Data types.String `tfsdk:"query"`
}

type discoveryViewResourceModel struct {
	ID                types.String                 `tfsdk:"id"`
	Name              types.String                 `tfsdk:"name"`
	Description       types.String                 `tfsdk:"description"`
	FilterData        *discoveryQueryResourceModel `tfsdk:"filter_data"`
	ExtraParameters   map[string]interface{}       `tfsdk:"extra_params"`
	Columns           types.List                   `tfsdk:"columns"`
	Sort              types.String                 `tfsdk:"sort"`
	GroupBy           types.List                   `tfsdk:"group_by"`
	GroupBy2          []groupByEntryModel          `tfsdk:"group_by_2"`
	OrganizationLevel types.Bool                   `tfsdk:"organization_level"`
	ViewType          types.String                 `tfsdk:"view_type"`
}

type groupByEntryModel struct {
	Key  types.String       `tfsdk:"key"`
	Sort []groupBySortModel `tfsdk:"sort"`
}

type groupBySortModel struct {
	Field     types.String `tfsdk:"field"`
	Direction types.String `tfsdk:"direction"`
}

// Keys within the view's extra_params object. columns2 holds the ordered
// display columns (under its "keys" array), sort2 holds the sort column, and
// groupBy2 holds the grouping columns. Each groupBy2 entry is an object with
// a required "key" and an optional "sort" (list of {field, direction}).
const (
	extraParamsColumnsKey     = "columns2"
	extraParamsSortKey        = "sort2"
	extraParamsGroupByKey     = "groupBy2"
	extraParamsDescriptionKey = "description"
)

// groupByAPIEntry mirrors the API shape of a single groupBy2 entry.
type groupByAPIEntry struct {
	Key  string
	Sort []groupBySortAPIEntry
}

type groupBySortAPIEntry struct {
	Field     string
	Direction string
}

// buildExtraParams converts the configured columns, sort, grouping and
// description into the extra_params shape the API expects:
// {"columns2": {"keys": [...]}, "sort2": "...", "groupBy2": [{"key": ..., "sort": [...]}, ...], "description": "..."}.
func buildExtraParams(columns []string, sort string, groupBy []groupByAPIEntry, description string) map[string]interface{} {
	extraParams := map[string]interface{}{}
	if len(columns) > 0 {
		extraParams[extraParamsColumnsKey] = map[string]interface{}{
			"keys": columns,
		}
	}
	if sort != "" {
		extraParams[extraParamsSortKey] = sort
	}
	if description != "" {
		extraParams[extraParamsDescriptionKey] = description
	}
	if len(groupBy) > 0 {
		entries := make([]map[string]interface{}, 0, len(groupBy))
		for _, entry := range groupBy {
			obj := map[string]interface{}{"key": entry.Key}
			if len(entry.Sort) > 0 {
				sortItems := make([]map[string]interface{}, 0, len(entry.Sort))
				for _, s := range entry.Sort {
					sortItems = append(sortItems, map[string]interface{}{
						"field":     s.Field,
						"direction": s.Direction,
					})
				}
				obj["sort"] = sortItems
			}
			entries = append(entries, obj)
		}
		extraParams[extraParamsGroupByKey] = entries
	}
	return extraParams
}

// extractSort pulls the sort column out of an API extra_params object.
func extractSort(extraParams map[string]interface{}) string {
	if sort, ok := extraParams[extraParamsSortKey].(string); ok {
		return sort
	}
	return ""
}

// extractDescription pulls the view description out of an API extra_params object.
func extractDescription(extraParams map[string]interface{}) string {
	if description, ok := extraParams[extraParamsDescriptionKey].(string); ok {
		return description
	}
	return ""
}

// extractGroupBy pulls the grouping columns out of an API extra_params object.
// The API may return either the legacy shape (list of strings) or the current
// shape (list of {key, sort} objects); both are normalized here.
func extractGroupBy(extraParams map[string]interface{}) []groupByAPIEntry {
	rawGroupBy, ok := extraParams[extraParamsGroupByKey].([]interface{})
	if !ok {
		return nil
	}
	groupBy := make([]groupByAPIEntry, 0, len(rawGroupBy))
	for _, raw := range rawGroupBy {
		if entry, ok := parseGroupByEntry(raw); ok {
			groupBy = append(groupBy, entry)
		}
	}
	return groupBy
}

// parseGroupByEntry normalizes a single groupBy2 element coming from the API
// into a groupByAPIEntry. It accepts both legacy string entries and the
// current object shape.
func parseGroupByEntry(raw interface{}) (groupByAPIEntry, bool) {
	switch value := raw.(type) {
	case string:
		return groupByAPIEntry{Key: value}, true
	case map[string]interface{}:
		entry := groupByAPIEntry{}
		if key, ok := value["key"].(string); ok {
			entry.Key = key
		}
		entry.Sort = parseGroupBySort(value["sort"])
		return entry, true
	}
	return groupByAPIEntry{}, false
}

// parseGroupBySort converts the raw "sort" payload of a groupBy2 entry into
// the typed slice. Unknown shapes and unknown items are skipped.
func parseGroupBySort(raw interface{}) []groupBySortAPIEntry {
	rawSort, ok := raw.([]interface{})
	if !ok || len(rawSort) == 0 {
		return nil
	}
	out := make([]groupBySortAPIEntry, 0, len(rawSort))
	for _, rawItem := range rawSort {
		item, ok := rawItem.(map[string]interface{})
		if !ok {
			continue
		}
		sortEntry := groupBySortAPIEntry{}
		if field, ok := item["field"].(string); ok {
			sortEntry.Field = field
		}
		if direction, ok := item["direction"].(string); ok {
			sortEntry.Direction = direction
		}
		out = append(out, sortEntry)
	}
	return out
}

// groupByOnlyKeys reports whether every entry has only a key (no sort), i.e.
// fits the legacy `group_by` (list of strings) shape.
func groupByOnlyKeys(groupBy []groupByAPIEntry) bool {
	for _, entry := range groupBy {
		if len(entry.Sort) > 0 {
			return false
		}
	}
	return true
}

// groupByKeys returns the bare keys for the legacy list-of-strings projection.
func groupByKeys(groupBy []groupByAPIEntry) []string {
	keys := make([]string, 0, len(groupBy))
	for _, entry := range groupBy {
		keys = append(keys, entry.Key)
	}
	return keys
}

// apiGroupByToModel converts API entries back into the typed resource model
// used by the `group_by_2` attribute.
func apiGroupByToModel(entries []groupByAPIEntry) []groupByEntryModel {
	if len(entries) == 0 {
		return nil
	}
	out := make([]groupByEntryModel, 0, len(entries))
	for _, entry := range entries {
		item := groupByEntryModel{Key: types.StringValue(entry.Key)}
		for _, s := range entry.Sort {
			item.Sort = append(item.Sort, groupBySortModel{
				Field:     types.StringValue(s.Field),
				Direction: types.StringValue(s.Direction),
			})
		}
		out = append(out, item)
	}
	return out
}

// planGroupByEntries builds the API entries from the resource model, preferring
// the new `group_by_2` attribute and falling back to the legacy `group_by`.
func planGroupByEntries(ctx context.Context, plan discoveryViewResourceModel) []groupByAPIEntry {
	if len(plan.GroupBy2) > 0 {
		entries := make([]groupByAPIEntry, 0, len(plan.GroupBy2))
		for _, item := range plan.GroupBy2 {
			entry := groupByAPIEntry{Key: item.Key.ValueString()}
			for _, s := range item.Sort {
				entry.Sort = append(entry.Sort, groupBySortAPIEntry{
					Field:     s.Field.ValueString(),
					Direction: s.Direction.ValueString(),
				})
			}
			entries = append(entries, entry)
		}
		return entries
	}
	keys := listToStrings(ctx, plan.GroupBy)
	if len(keys) == 0 {
		return nil
	}
	entries := make([]groupByAPIEntry, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, groupByAPIEntry{Key: key})
	}
	return entries
}

// extractColumns pulls the ordered column list out of an API extra_params
// object. Newer (v2-migrated) views store the display columns under
// extra_params.columns2.keys; older / UI-migrated views instead carry the full
// column catalog under extra_params.table_config.columns with the hidden ones
// listed in table_config.hiddenKeys. We prefer columns2 and fall back to
// table_config so imported views round-trip cleanly either way.
func extractColumns(extraParams map[string]interface{}) []string {
	if columnsConfig, ok := extraParams[extraParamsColumnsKey].(map[string]interface{}); ok {
		if rawKeys, ok := columnsConfig["keys"].([]interface{}); ok {
			keys := make([]string, 0, len(rawKeys))
			for _, rawKey := range rawKeys {
				if key, ok := rawKey.(string); ok {
					keys = append(keys, key)
				}
			}
			if len(keys) > 0 {
				return keys
			}
		}
	}
	return extractTableConfigColumns(extraParams)
}

// extractTableConfigColumns returns the visible column keys from the legacy
// table_config shape: every table_config.columns[].key that is not listed in
// table_config.hiddenKeys, preserving order. This mirrors the visible-column
// set the Orca UI exports to HCL.
func extractTableConfigColumns(extraParams map[string]interface{}) []string {
	tableConfig, ok := extraParams["table_config"].(map[string]interface{})
	if !ok {
		return nil
	}
	rawColumns, ok := tableConfig["columns"].([]interface{})
	if !ok {
		return nil
	}
	hidden := map[string]bool{}
	if rawHidden, ok := tableConfig["hiddenKeys"].([]interface{}); ok {
		for _, h := range rawHidden {
			if key, ok := h.(string); ok {
				hidden[key] = true
			}
		}
	}
	keys := make([]string, 0, len(rawColumns))
	for _, rawColumn := range rawColumns {
		column, ok := rawColumn.(map[string]interface{})
		if !ok {
			continue
		}
		key, ok := column["key"].(string)
		if !ok || key == "" || hidden[key] {
			continue
		}
		keys = append(keys, key)
	}
	return keys
}

func listToStrings(ctx context.Context, list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var out []string
	list.ElementsAs(ctx, &out, false)
	return out
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
			"description": schema.StringAttribute{
				Description: "Optional free-text description of the discovery view (shown in the Orca UI). " +
					"Stored under the view's `extra_params.description`.",
				Optional: true,
			},
			"view_type": schema.StringAttribute{
				Description: "Should be set to 'discovery' for discovery views.",
				Required:    true,
			},
			"organization_level": schema.BoolAttribute{
				Description: "If set to true, it is a shared discovery view (visible to every member of your Orca org). " +
					"If set to false, it is a personal discovery view that is **scoped to the user identity behind the API token used by this provider**, not whichever user is logged into the Orca UI. " +
					"Personal views created via Terraform therefore only appear in the UI when you log in as that token user; if your TF token user differs from your UI user, the view will exist in the API (`GET /api/user_preferences?view_type=discovery` returns it under `data.user_preferences[]`) but it will not be shown in the UI. " +
					"For views that should be visible to multiple users, set `organization_level = true`.",
				Required: true,
				Validators: []validator.Bool{
					personalViewWarningValidator{},
				},
			},
			"extra_params": schema.MapAttribute{
				Description: "Reserved for additional view parameters. To control which columns are displayed, use the `columns` attribute instead.",
				ElementType: types.StringType,
				Required:    true,
			},
			"columns": schema.ListAttribute{
				Description: "Ordered list of columns to display in the discovery view. When omitted, the view uses Orca's default columns. " +
					"Each entry is either a Sonar field name (e.g. `OrcaScore`, `CloudAccount`, `Exposure`, `SensitiveData`, `Tags`, `AssetUniqueId`) " +
					"or a special aggregate column (e.g. `$overview`, `$alertsStats`, `$attackPaths`, `$targetAttackPaths`). " +
					"Valid keys depend on the models targeted by `filter_data.query`. " +
					"The authoritative way to obtain exact keys for a given view is to configure the columns in the Orca UI and read them back from " +
					"`GET /api/user_preferences/{id}?view_type=discovery` (`data.extra_params.columns2.keys`). See the resource documentation for details.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"sort": schema.StringAttribute{
				Description: "Column to sort the view by. Use a Sonar field name; prefix with `-` for descending order (e.g. `-OrcaScore`). " +
					"When omitted, the view uses Orca's default sort.",
				Optional: true,
			},
			"group_by": schema.ListAttribute{
				Description: "Ordered list of columns to group the view results by (e.g. `AlertType`). When omitted, results are not grouped. " +
					"Deprecated: use `group_by_2` instead, which supports per-group sorting. Only one of `group_by` and `group_by_2` may be set.",
				ElementType:        types.StringType,
				Optional:           true,
				DeprecationMessage: "Use `group_by_2` instead. `group_by` does not support per-group sorting and will be removed in a future release.",
			},
			"group_by_2": schema.ListNestedAttribute{
				Description: "Ordered list of group-by entries. Each entry has a `key` (the column to group by, e.g. `CloudAccount.Name`) and an optional `sort` " +
					"list that controls the per-group ordering (each sort item has a `field` such as `COUNT` and a `direction` of `asc` or `desc`). " +
					"Mutually exclusive with the deprecated `group_by` attribute.",
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Description: "Column key to group by (e.g. `CloudAccount.Name`, `AlertType`).",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
						"sort": schema.ListNestedAttribute{
							Description: "Optional per-group sort. Each entry sorts the group by `field` in `direction`.",
							Optional:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"field": schema.StringAttribute{
										Description: "Field to sort the group by (e.g. `COUNT`).",
										Required:    true,
										Validators: []validator.String{
											stringvalidator.LengthAtLeast(1),
										},
									},
									"direction": schema.StringAttribute{
										Description: "Sort direction: `asc` or `desc`.",
										Required:    true,
										Validators: []validator.String{
											stringvalidator.OneOf("asc", "desc"),
										},
									},
								},
							},
						},
					},
				},
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

func (r *discoveryViewResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("group_by"),
			path.MatchRoot("group_by_2"),
		),
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
		ExtraParameters:   buildExtraParams(listToStrings(ctx, plan.Columns), plan.Sort.ValueString(), planGroupByEntries(ctx, plan), plan.Description.ValueString()),
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
	if instance == nil {
		resp.Diagnostics.AddError(
			"Error refreshing Discovery view",
			"Could not read discovery view after create: not found",
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
			errReadingDiscoveryView,
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
			errReadingDiscoveryView,
			fmt.Sprintf("Could not read discovery view ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}
	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("Discovery view %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	populateDiscoveryViewState(ctx, &state, instance, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// populateDiscoveryViewState maps the API instance onto the model. Split out of
// Read to keep that function's cognitive complexity low.
func populateDiscoveryViewState(ctx context.Context, state *discoveryViewResourceModel, instance *api_client.DiscoveryView, resp *resource.ReadResponse) {
	// Prefer the query already in state to avoid churn from JSON key
	// reordering on normal refresh. On import there is no prior state, so
	// derive the query from the API response instead.
	queryString := ""
	if state.FilterData != nil {
		queryString = state.FilterData.Data.ValueString()
	}
	if queryString == "" && len(instance.FilterData.Data) > 0 {
		queryBytes, err := json.Marshal(instance.FilterData.Data)
		if err != nil {
			resp.Diagnostics.AddError(
				errReadingDiscoveryView,
				fmt.Sprintf("Could not marshal filter_data query for ID %s: %s", state.ID.ValueString(), err.Error()),
			)
			return
		}
		queryString = string(queryBytes)
	}

	state.ID = types.StringValue(instance.ID)
	state.Name = types.StringValue(instance.Name)
	state.OrganizationLevel = types.BoolValue(instance.OrganizationLevel)
	state.ViewType = types.StringValue(instance.ViewType)
	state.ExtraParameters = make(map[string]interface{})
	state.FilterData = &discoveryQueryResourceModel{Data: types.StringValue(queryString)}

	columns := extractColumns(instance.ExtraParameters)
	if len(columns) > 0 {
		columnsList, columnsDiags := types.ListValueFrom(ctx, types.StringType, columns)
		resp.Diagnostics.Append(columnsDiags...)
		state.Columns = columnsList
	} else {
		state.Columns = types.ListNull(types.StringType)
	}

	if sort := extractSort(instance.ExtraParameters); sort != "" {
		state.Sort = types.StringValue(sort)
	} else {
		state.Sort = types.StringNull()
	}

	if description := extractDescription(instance.ExtraParameters); description != "" {
		state.Description = types.StringValue(description)
	} else {
		state.Description = types.StringNull()
	}

	applyGroupByToState(ctx, state, instance, resp)
}

// applyGroupByToState decides which group-by attribute to populate. It
// preserves the user's chosen attribute when possible: if state already used
// the legacy `group_by` and the data still fits that shape (no per-group sort),
// keep `group_by`. Otherwise use `group_by_2`.
func applyGroupByToState(ctx context.Context, state *discoveryViewResourceModel, instance *api_client.DiscoveryView, resp *resource.ReadResponse) {
	groupBy := extractGroupBy(instance.ExtraParameters)
	legacyGroupByInUse := !state.GroupBy.IsNull() && !state.GroupBy.IsUnknown()
	switch {
	case len(groupBy) == 0:
		state.GroupBy = types.ListNull(types.StringType)
		state.GroupBy2 = nil
	case legacyGroupByInUse && groupByOnlyKeys(groupBy):
		groupByList, groupByDiags := types.ListValueFrom(ctx, types.StringType, groupByKeys(groupBy))
		resp.Diagnostics.Append(groupByDiags...)
		state.GroupBy = groupByList
		state.GroupBy2 = nil
	default:
		state.GroupBy = types.ListNull(types.StringType)
		state.GroupBy2 = apiGroupByToModel(groupBy)
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
		ExtraParameters:   buildExtraParams(listToStrings(ctx, plan.Columns), plan.Sort.ValueString(), planGroupByEntries(ctx, plan), plan.Description.ValueString()),
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
	if instance == nil {
		resp.Diagnostics.AddError(
			"Error updating discovery view",
			"Could not read discovery view after update: not found",
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
