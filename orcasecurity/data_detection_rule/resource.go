package data_detection_rule

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                     = &dataDetectionRuleResource{}
	_ resource.ResourceWithConfigure        = &dataDetectionRuleResource{}
	_ resource.ResourceWithImportState      = &dataDetectionRuleResource{}
	_ resource.ResourceWithConfigValidators = &dataDetectionRuleResource{}
)

type dataDetectionRuleResource struct {
	apiClient *api_client.APIClient
}

type stateModel struct {
	ID                    types.String `tfsdk:"id"`
	OrganizationID        types.String `tfsdk:"organization_id"`
	Name                  types.String `tfsdk:"name"`
	Priority              types.Int64  `tfsdk:"priority"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	Action                types.String `tfsdk:"action"`
	Feature               types.String `tfsdk:"feature"`
	Policies              types.List   `tfsdk:"policies"`
	SelectorCloudAccounts types.List   `tfsdk:"selector_cloud_accounts"`
	SelectorBusinessUnits types.List   `tfsdk:"selector_business_units"`
	Tags                  types.List   `tfsdk:"tags"`
	IsDefaultRule         types.Bool   `tfsdk:"is_default_rule"`
}

func NewDataDetectionRuleResource() resource.Resource {
	return &dataDetectionRuleResource{}
}

func (r *dataDetectionRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_data_detection_rule"
}

func (r *dataDetectionRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

// ConfigValidators enforces the server-side scope constraint at plan time:
// at least one of selector_cloud_accounts / selector_business_units / tags
// must be set.
func (r *dataDetectionRuleResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("selector_cloud_accounts"),
			path.MatchRoot("selector_business_units"),
			path.MatchRoot("tags"),
		),
	}
}

func (r *dataDetectionRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *dataDetectionRuleResource) Schema(_ context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Provides a DSPM data detection rule (scan configuration rule with feature `DSPM Scanning`). A rule binds data protection policies to a scan scope.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Rule ID.",
			},
			"organization_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Orca organization ID.",
			},
			"name": schema.StringAttribute{
				Description: "Rule name. Must be unique within the organization.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"priority": schema.Int64Attribute{
				Description: "Rule priority (unique per organization). If omitted, the server auto-assigns the next free priority.",
				Optional:    true,
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the rule is enabled. Defaults to `false` (matches the Orca UI default for new rules).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"action": schema.StringAttribute{
				Description: "Rule action. Valid values are `scan` and `do_not_scan`. Defaults to `scan`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("scan"),
				Validators: []validator.String{
					stringvalidator.OneOf("scan", "do_not_scan"),
				},
			},
			"feature": schema.StringAttribute{
				Description: "Scan feature. Defaults to `DSPM Scanning`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("DSPM Scanning"),
			},
			"policies": schema.ListAttribute{
				Description: "DSPM policy IDs (UUIDs) attached to the rule. Only valid when `feature` is `DSPM Scanning`.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"selector_cloud_accounts": schema.ListAttribute{
				Description: "Cloud account IDs in scope. At least one of `selector_cloud_accounts`, `selector_business_units`, or `tags` must be set.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"selector_business_units": schema.ListAttribute{
				Description: "Business unit IDs in scope. At least one of `selector_cloud_accounts`, `selector_business_units`, or `tags` must be set.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"tags": schema.ListAttribute{
				Description: "Rule tags (also used for scoping). At least one of `selector_cloud_accounts`, `selector_business_units`, or `tags` must be set.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"is_default_rule": schema.BoolAttribute{
				Description: "Whether this is an Orca-managed default rule. Always false for Terraform-created rules.",
				Computed:    true,
			},
		},
	}
}

// stringListToAPI converts a types.List of strings to a Go slice.
// Null and unknown lists become nil (omitted from the JSON payload).
func stringListToAPI(ctx context.Context, list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var out []string
	_ = list.ElementsAs(ctx, &out, false)
	return out
}

// stringListFromAPIPreserveNull maps an API string slice back to state.
// When the API returns empty and the prior state was null (attribute not
// configured), null is preserved to avoid a perpetual null-vs-[] diff.
func stringListFromAPIPreserveNull(ctx context.Context, prior types.List, values []string) (types.List, diag.Diagnostics) {
	if len(values) == 0 && prior.IsNull() {
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueFrom(ctx, types.StringType, values)
}

func int64ToAPIPtr(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	value := v.ValueInt64()
	return &value
}

func int64FromAPIPtr(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}

func generateRulePayload(ctx context.Context, plan stateModel) api_client.DataDetectionRule {
	return api_client.DataDetectionRule{
		Name:                  plan.Name.ValueString(),
		Feature:               plan.Feature.ValueString(),
		Action:                plan.Action.ValueString(),
		Priority:              int64ToAPIPtr(plan.Priority),
		Enabled:               plan.Enabled.ValueBool(),
		SelectorCloudAccounts: stringListToAPI(ctx, plan.SelectorCloudAccounts),
		SelectorBusinessUnits: stringListToAPI(ctx, plan.SelectorBusinessUnits),
		Tags:                  stringListToAPI(ctx, plan.Tags),
		Policies:              stringListToAPI(ctx, plan.Policies),
	}
}

func (r *dataDetectionRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan stateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// non-standard REST: create is PUT on the collection, returns data.rule_id
	ruleID, err := r.apiClient.CreateDataDetectionRule(generateRulePayload(ctx, plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Data Detection Rule",
			"Could not create Data Detection Rule, unexpected error: "+err.Error(),
		)
		return
	}

	// the create response only carries the id — refresh to resolve the
	// server-assigned priority, organization and default flag
	instance, err := r.apiClient.GetDataDetectionRule(ruleID)
	if err != nil || instance == nil {
		message := "rule vanished right after creation"
		if err != nil {
			message = err.Error()
		}
		resp.Diagnostics.AddError(
			"Error refreshing Data Detection Rule",
			fmt.Sprintf("Could not read Data Detection Rule ID %s after create: %s", ruleID, message),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.OrganizationID = types.StringValue(instance.OrganizationID)
	plan.Priority = int64FromAPIPtr(instance.Priority)
	plan.IsDefaultRule = types.BoolValue(instance.IsDefaultRule)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *dataDetectionRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.GetDataDetectionRule(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Data Detection Rule",
			fmt.Sprintf("Could not read Data Detection Rule ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}
	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("Data Detection Rule %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	policies, d := stringListFromAPIPreserveNull(ctx, state.Policies, instance.Policies)
	resp.Diagnostics.Append(d...)
	cloudAccounts, d := stringListFromAPIPreserveNull(ctx, state.SelectorCloudAccounts, instance.SelectorCloudAccounts)
	resp.Diagnostics.Append(d...)
	businessUnits, d := stringListFromAPIPreserveNull(ctx, state.SelectorBusinessUnits, instance.SelectorBusinessUnits)
	resp.Diagnostics.Append(d...)
	tags, d := stringListFromAPIPreserveNull(ctx, state.Tags, instance.Tags)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(instance.ID)
	state.OrganizationID = types.StringValue(instance.OrganizationID)
	state.Name = types.StringValue(instance.Name)
	state.Priority = int64FromAPIPtr(instance.Priority)
	state.Enabled = types.BoolValue(instance.Enabled)
	state.Action = types.StringValue(instance.Action)
	state.Feature = types.StringValue(instance.Feature)
	state.Policies = policies
	state.SelectorCloudAccounts = cloudAccounts
	state.SelectorBusinessUnits = businessUnits
	state.Tags = tags
	state.IsDefaultRule = types.BoolValue(instance.IsDefaultRule)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *dataDetectionRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan stateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state stateModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"ID is null",
			"Could not update Data Detection Rule, unexpected error: missing ID",
		)
		return
	}

	// non-standard REST: update goes through POST /bulk_rules
	updateReq := generateRulePayload(ctx, plan)
	updateReq.ID = state.ID.ValueString()
	if err := r.apiClient.UpdateDataDetectionRule(updateReq); err != nil {
		resp.Diagnostics.AddError(
			"Error updating Data Detection Rule",
			"Could not update Data Detection Rule, unexpected error: "+err.Error(),
		)
		return
	}

	instance, err := r.apiClient.GetDataDetectionRule(state.ID.ValueString())
	if err != nil || instance == nil {
		message := "rule vanished right after update"
		if err != nil {
			message = err.Error()
		}
		resp.Diagnostics.AddError(
			"Error refreshing Data Detection Rule",
			fmt.Sprintf("Could not read Data Detection Rule ID %s after update: %s", state.ID.ValueString(), message),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.OrganizationID = types.StringValue(instance.OrganizationID)
	plan.Priority = int64FromAPIPtr(instance.Priority)
	plan.IsDefaultRule = types.BoolValue(instance.IsDefaultRule)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *dataDetectionRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteDataDetectionRule(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Data Detection Rule",
			"Could not delete Data Detection Rule, unexpected error: "+err.Error(),
		)
	}
}
