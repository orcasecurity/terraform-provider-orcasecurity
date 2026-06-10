package custom_tag_rule

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.Resource                = &customTagRuleResource{}
	_ resource.ResourceWithConfigure   = &customTagRuleResource{}
	_ resource.ResourceWithImportState = &customTagRuleResource{}
)

type customTagRuleResource struct {
	apiClient *api_client.APIClient
}

type stateModel struct {
	ID          types.String      `tfsdk:"id"`
	Name        types.String      `tfsdk:"name"`
	Description types.String      `tfsdk:"description"`
	Tags        map[string]string `tfsdk:"tags"`
	Rule        types.String      `tfsdk:"rule"`
	RuleType    types.String      `tfsdk:"rule_type"`
	Disabled    types.Bool        `tfsdk:"disabled"`
}

func NewCustomTagRuleResource() resource.Resource {
	return &customTagRuleResource{}
}

func (r *customTagRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_tag_rule"
}

func (r *customTagRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *customTagRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *customTagRuleResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a custom tag rule. Custom tag rules automatically apply custom tags to all assets that match a discovery (Sonar) query.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Custom tag rule ID.",
			},
			"name": schema.StringAttribute{
				Description: "Custom tag rule name. Must be unique within the organization.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: "Custom tag rule description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"tags": schema.MapAttribute{
				Description: "The tags (key/value pairs) to apply to assets matching the rule. " +
					"Each tag key must be unique across all custom tag rules in the organization.",
				ElementType: types.StringType,
				Required:    true,
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
				},
			},
			"rule": schema.StringAttribute{
				Description: "The discovery (Sonar) query that selects the assets to tag. " +
					"A query string when `rule_type` is `string`, or a JSON-encoded query when `rule_type` is `json`.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"rule_type": schema.StringAttribute{
				Description: "Rule format. Valid values are `string` and `json`. Defaults to `string`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(api_client.CustomTagRuleRuleTypeString),
				Validators: []validator.String{
					stringvalidator.OneOf(api_client.CustomTagRuleRuleTypeString, api_client.CustomTagRuleRuleTypeJSON),
				},
			},
			"disabled": schema.BoolAttribute{
				Description: "Whether the rule is disabled. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func generateAPIRequest(plan stateModel) api_client.CustomTagRule {
	return api_client.CustomTagRule{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Tags:        plan.Tags,
		Rule:        plan.Rule.ValueString(),
		RuleType:    plan.RuleType.ValueString(),
		Disabled:    plan.Disabled.ValueBool(),
	}
}

func (r *customTagRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan stateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.CreateCustomTagRule(generateAPIRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating custom tag rule",
			"Could not create custom tag rule, unexpected error: "+err.Error(),
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

func (r *customTagRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.GetCustomTagRule(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading custom tag rule",
			fmt.Sprintf("Could not read custom tag rule ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("Custom tag rule %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(instance.Name)
	state.Description = types.StringValue(instance.Description)
	state.Tags = instance.Tags
	state.RuleType = types.StringValue(instance.RuleType)
	state.Disabled = types.BoolValue(instance.Disabled)

	// For JSON rules, the API stores a re-serialized version of the rule.
	// Keep the state value when it is semantically equal to avoid
	// formatting-only diffs.
	if instance.RuleType == api_client.CustomTagRuleRuleTypeJSON &&
		!state.Rule.IsNull() &&
		jsonEqual(state.Rule.ValueString(), instance.Rule) {
		// keep current state value
	} else {
		state.Rule = types.StringValue(instance.Rule)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customTagRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan stateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.apiClient.UpdateCustomTagRule(plan.ID.ValueString(), generateAPIRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating custom tag rule",
			"Could not update custom tag rule, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customTagRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteCustomTagRule(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting custom tag rule",
			"Could not delete custom tag rule, unexpected error: "+err.Error(),
		)
		return
	}
}

func jsonEqual(a, b string) bool {
	var aValue, bValue interface{}
	if err := json.Unmarshal([]byte(a), &aValue); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &bValue); err != nil {
		return false
	}
	return reflect.DeepEqual(aValue, bValue)
}
