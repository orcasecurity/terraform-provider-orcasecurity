package custom_discovery_alert

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/alert_common"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                     = &customDiscoveryAlertResource{}
	_ resource.ResourceWithConfigure        = &customDiscoveryAlertResource{}
	_ resource.ResourceWithImportState      = &customDiscoveryAlertResource{}
	_ resource.ResourceWithConfigValidators = &customDiscoveryAlertResource{}
)

const errReadingAlert = "Error reading Alert"

type customDiscoveryAlertResource struct {
	apiClient *api_client.APIClient
}

type remediationTextStateModel struct {
	Enable types.Bool   `tfsdk:"enable"`
	Text   types.String `tfsdk:"text"`
}

type stateModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	// Rule            types.String               `tfsdk:"rule"` //Even though it's in the API response (as a null value), this can be safely commented because it's not a required field to create the custom alert.
	RuleType        types.String               `tfsdk:"rule_type"`
	OrganizationID  types.String               `tfsdk:"organization_id"`
	Category        types.String               `tfsdk:"category"`
	RuleJson        types.String               `tfsdk:"rule_json"`
	OrcaScore       types.Float64              `tfsdk:"orca_score"`
	ContextScore    types.Bool                 `tfsdk:"context_score"`
	Frameworks      types.List                 `tfsdk:"compliance_frameworks"`
	RemediationText *remediationTextStateModel `tfsdk:"remediation_text"`
}

func NewCustomDiscoveryAlertResource() resource.Resource {
	return &customDiscoveryAlertResource{}
}

func (r *customDiscoveryAlertResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_custom_discovery_alert"
}

func (r *customDiscoveryAlertResource) Configure(_ context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *customDiscoveryAlertResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{}
}

func (r *customDiscoveryAlertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *customDiscoveryAlertResource) Schema(_ context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Provides a custom discovery-based alert.",
		Attributes: alert_common.MergeAttributes(map[string]schema.Attribute{
			"rule_type": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Custom alert rule type (unique, Orca-computed identifier).",
			},
			"rule_json": schema.StringAttribute{
				Description: "The discovery query (JSON) used to define the rule.",
				Optional:    true,
			},
			"orca_score": schema.Float64Attribute{
				Description: "The base score of the alert.",
				Required:    true,
			},
			"context_score": schema.BoolAttribute{
				Description: "Allows Orca to adjust the score using asset context.",
				Required:    true,
			},
			"remediation_text":      alert_common.RemediationTextAttribute(),
			"compliance_frameworks": alert_common.ComplianceFrameworksAttribute(),
		}),
	}
}

func (r *customDiscoveryAlertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan stateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateCategory(r.apiClient, plan.Category.ValueString()); err != nil {
		resp.Diagnostics.AddError("Invalid category", err.Error())
		return
	}

	//Generate API request body from plan
	queryString := plan.RuleJson.ValueString()
	query := make(map[string]interface{})
	err := json.Unmarshal([]byte(queryString), &query)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating custom alert",
			"Could not create custom alert, unexpected error: "+err.Error(),
		)
		return
	}

	createFrameworks, frameworkDiags := alert_common.FrameworksRequest(ctx, plan.Frameworks)
	resp.Diagnostics.Append(frameworkDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := api_client.CustomDiscoveryAlert{
		Name:                 plan.Name.ValueString(),
		RuleJson:             query,
		Description:          plan.Description.ValueString(),
		RuleType:             plan.RuleType.ValueString(),
		Category:             plan.Category.ValueString(),
		OrcaScore:            plan.OrcaScore.ValueFloat64(),
		ContextScore:         plan.ContextScore.ValueBool(),
		ComplianceFrameworks: createFrameworks,
	}

	if plan.RemediationText != nil {
		createReq.RemediationText = &api_client.CustomDiscoveryAlertRemediationText{
			AlertType: "", // available only after alert creation
			Enable:    plan.RemediationText.Enable.ValueBool(),
			Text:      plan.RemediationText.Text.ValueString(),
		}
	}

	instance, err := r.apiClient.CreateCustomDiscoveryAlert(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Alert",
			"Could not create Alert, unexpected error: "+err.Error(),
		)
		return
	}

	instance, err = r.apiClient.GetCustomDiscoveryAlert(instance.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error refreshing Alert",
			"Could not create Alert, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.RuleType = types.StringValue(instance.RuleType)
	plan.OrganizationID = types.StringValue(instance.OrganizationID)

	frameworks, frameworkDiags := alert_common.FrameworksAfterCreate(ctx, plan.Frameworks, instance.ComplianceFrameworks)
	resp.Diagnostics.Append(frameworkDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Frameworks = frameworks

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customDiscoveryAlertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.apiClient.DoesCustomDiscoveryAlertExist(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			errReadingAlert,
			fmt.Sprintf("Could not read Alert ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	if !exists {
		tflog.Warn(ctx, fmt.Sprintf("Alert %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	instance, err := r.apiClient.GetCustomDiscoveryAlert(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			errReadingAlert,
			fmt.Sprintf("Could not read Alert ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	state.ID = types.StringValue(instance.ID)
	state.Name = types.StringValue(instance.Name)
	state.Description = types.StringValue(instance.Description)
	state.RuleType = types.StringValue(instance.RuleType)
	state.OrganizationID = types.StringValue(instance.OrganizationID)
	state.Category = types.StringValue(instance.Category)
	state.ContextScore = types.BoolValue(instance.ContextScore)
	state.OrcaScore = types.Float64Value(instance.OrcaScore)

	// Prefer the rule_json already in state to avoid churn from JSON key
	// reordering on normal refresh. On import there is no prior state, so
	// derive it from the API response instead.
	ruleJsonString := state.RuleJson.ValueString()
	if ruleJsonString == "" && len(instance.RuleJson) > 0 {
		ruleJsonBytes, err := json.Marshal(instance.RuleJson)
		if err != nil {
			resp.Diagnostics.AddError(
				errReadingAlert,
				fmt.Sprintf("Could not marshal rule_json for ID %s: %s", state.ID.ValueString(), err.Error()),
			)
			return
		}
		ruleJsonString = string(ruleJsonBytes)
	}
	if ruleJsonString != "" {
		state.RuleJson = types.StringValue(ruleJsonString)
	} else {
		state.RuleJson = types.StringNull()
	}

	if instance.RemediationText.Text != "" {
		state.RemediationText = &remediationTextStateModel{
			Enable: types.BoolValue(instance.RemediationText.Enable),
			Text:   types.StringValue(instance.RemediationText.Text),
		}
	}

	frameworks, diags := alert_common.FrameworksState(ctx, instance.ComplianceFrameworks)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Frameworks = frameworks

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customDiscoveryAlertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan stateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state stateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"ID is null",
			"Could not update Alert, unexpected error: "+plan.ID.ValueString(),
		)
		return
	}

	if err := validateCategory(r.apiClient, plan.Category.ValueString()); err != nil {
		resp.Diagnostics.AddError("Invalid category", err.Error())
		return
	}

	//Generate API request body from plan
	queryString := plan.RuleJson.ValueString()
	query := make(map[string]interface{})
	err := json.Unmarshal([]byte(queryString), &query)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating custom alert",
			"Could not update custom alert, unexpected error: "+err.Error(),
		)
		return
	}

	updateFrameworks, frameworkDiags := alert_common.FrameworksRequest(ctx, plan.Frameworks)
	resp.Diagnostics.Append(frameworkDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := api_client.CustomDiscoveryAlert{
		Name:                 plan.Name.ValueString(),
		Description:          plan.Description.ValueString(),
		RuleJson:             query,
		RuleType:             plan.RuleType.ValueString(),
		OrcaScore:            plan.OrcaScore.ValueFloat64(),
		ContextScore:         plan.ContextScore.ValueBool(),
		Category:             plan.Category.ValueString(),
		OrganizationID:       plan.OrganizationID.ValueString(),
		ComplianceFrameworks: updateFrameworks,
	}

	if plan.RemediationText != nil {
		updateReq.RemediationText = &api_client.CustomDiscoveryAlertRemediationText{
			AlertType: plan.RuleType.ValueString(),
			Enable:    plan.RemediationText.Enable.ValueBool(),
			Text:      plan.RemediationText.Text.ValueString(),
		}
	}

	clearedButFailed, err := alert_common.ReplaceFrameworks(state.Frameworks, plan.Frameworks, updateReq,
		func(request *api_client.CustomDiscoveryAlert) { request.ComplianceFrameworks = nil },
		func(request api_client.CustomDiscoveryAlert) error {
			_, err := r.apiClient.UpdateCustomDiscoveryAlert(plan.ID.ValueString(), request)
			return err
		},
	)
	if err != nil {
		alert_common.ReportFrameworkWriteFailure(ctx, resp, &plan, &plan.Frameworks, clearedButFailed, err)
		return
	}

	// if instance.RemediationText == nil || instance.RemediationText.Text == "" {
	// 	plan.RemediationText = nil
	// }

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customDiscoveryAlertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteCustomDiscoveryAlert(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Alert",
			"Could not delete Alert, unexpected error: "+err.Error(),
		)
		return
	}
}

func validateCategory(client *api_client.APIClient, category string) error {
	categories, err := client.GetAlertCategories()
	if err != nil {
		return err
	}

	for _, knownCategory := range categories {
		if knownCategory == category {
			return nil
		}
	}

	sort.Strings(categories)
	categoryValues := strings.Join(categories, ", ")
	return fmt.Errorf("invalid category. Please choose from: %s", categoryValues)
}
