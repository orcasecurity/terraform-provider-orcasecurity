package custom_sonar_alert

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/alert_common"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                     = &customSonarAlertResource{}
	_ resource.ResourceWithConfigure        = &customSonarAlertResource{}
	_ resource.ResourceWithImportState      = &customSonarAlertResource{}
	_ resource.ResourceWithConfigValidators = &customSonarAlertResource{}
)

type customSonarAlertResource struct {
	apiClient *api_client.APIClient
}

type remediationTextStateModel struct {
	Enable types.Bool   `tfsdk:"enable"`
	Text   types.String `tfsdk:"text"`
}

type stateModel struct {
	ID              types.String               `tfsdk:"id"`
	Name            types.String               `tfsdk:"name"`
	Description     types.String               `tfsdk:"description"`
	Rule            types.String               `tfsdk:"rule"`
	RuleType        types.String               `tfsdk:"rule_type"`
	OrganizationID  types.String               `tfsdk:"organization_id"`
	Category        types.String               `tfsdk:"category"`
	OrcaScore       types.Float64              `tfsdk:"orca_score"`
	ContextScore    types.Bool                 `tfsdk:"context_score"`
	Enabled         types.Bool                 `tfsdk:"enabled"`
	Frameworks      types.List                 `tfsdk:"compliance_frameworks"`
	RemediationText *remediationTextStateModel `tfsdk:"remediation_text"`
}

func NewCustomSonarAlertResource() resource.Resource {
	return &customSonarAlertResource{}
}

func (r *customSonarAlertResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_custom_sonar_alert"
}

func (r *customSonarAlertResource) Configure(_ context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *customSonarAlertResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{}
}

func (r *customSonarAlertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *customSonarAlertResource) Schema(_ context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Provides a custom sonar-based alert.",
		Attributes: alert_common.MergeAttributes(map[string]schema.Attribute{
			"rule": schema.StringAttribute{
				Description: "Sonar query that defines the rule.",
				Required:    true,
			},
			"rule_type": schema.StringAttribute{
				Description: "Custom alert rule type (unique, Orca-computed identifier).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"orca_score": schema.Float64Attribute{
				Description: "Alert score.",
				Required:    true,
			},
			"context_score": schema.BoolAttribute{
				Description: "Allow Orca to adjust the score using asset context.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the alert is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"remediation_text":      alert_common.RemediationTextAttribute(),
			"compliance_frameworks": alert_common.ComplianceFrameworksAttribute(),
		}),
	}
}

func (r *customSonarAlertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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

	// Default enabled to true if not specified
	enabled := true
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		enabled = plan.Enabled.ValueBool()
	}

	createFrameworks, frameworkDiags := alert_common.FrameworksRequest(ctx, plan.Frameworks)
	resp.Diagnostics.Append(frameworkDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := api_client.CustomAlert{
		Name:                 plan.Name.ValueString(),
		Description:          plan.Description.ValueString(),
		Rule:                 plan.Rule.ValueString(),
		RuleType:             plan.RuleType.ValueString(),
		Category:             plan.Category.ValueString(),
		OrcaScore:            plan.OrcaScore.ValueFloat64(),
		ContextScore:         plan.ContextScore.ValueBool(),
		Enabled:              enabled,
		ComplianceFrameworks: createFrameworks,
	}
	if plan.RemediationText != nil {
		createReq.RemediationText = &api_client.CustomSonarAlertRemediationText{
			AlertType: "", // available only after alert creation
			Enable:    plan.RemediationText.Enable.ValueBool(),
			Text:      plan.RemediationText.Text.ValueString(),
		}
	}

	instance, err := r.apiClient.CreateCustomSonarAlert(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Alert",
			"Could not create Alert, unexpected error: "+err.Error(),
		)
		return
	}

	createdID := instance.ID
	instance, err = r.apiClient.GetCustomSonarAlert(createdID)
	if err == nil && instance == nil {
		err = fmt.Errorf("alert %s not found right after creation", createdID)
	}
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
	plan.Enabled = types.BoolValue(instance.Enabled)

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

func (r *customSonarAlertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.GetCustomSonarAlert(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Alert",
			fmt.Sprintf("Could not read Alert ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("Alert %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(instance.ID)
	state.Name = types.StringValue(instance.Name)
	state.Description = types.StringValue(instance.Description)
	state.Rule = types.StringValue(instance.Rule)
	state.RuleType = types.StringValue(instance.RuleType)
	state.OrganizationID = types.StringValue(instance.OrganizationID)
	state.Category = types.StringValue(instance.Category)
	state.ContextScore = types.BoolValue(instance.ContextScore)
	state.OrcaScore = types.Float64Value(instance.OrcaScore)
	state.Enabled = types.BoolValue(instance.Enabled)

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

func (r *customSonarAlertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	// If enabled is not specified in the plan, preserve the existing state value
	enabled := state.Enabled.ValueBool()
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		enabled = plan.Enabled.ValueBool()
	}

	updateFrameworks, frameworkDiags := alert_common.FrameworksRequest(ctx, plan.Frameworks)
	resp.Diagnostics.Append(frameworkDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := api_client.CustomAlert{
		Name:                 plan.Name.ValueString(),
		Description:          plan.Description.ValueString(),
		Rule:                 plan.Rule.ValueString(),
		RuleType:             plan.RuleType.ValueString(),
		OrcaScore:            plan.OrcaScore.ValueFloat64(),
		ContextScore:         plan.ContextScore.ValueBool(),
		Enabled:              enabled,
		Category:             plan.Category.ValueString(),
		OrganizationID:       plan.OrganizationID.ValueString(),
		ComplianceFrameworks: updateFrameworks,
	}

	if plan.RemediationText != nil {
		updateReq.RemediationText = &api_client.CustomSonarAlertRemediationText{
			AlertType: plan.RuleType.ValueString(),
			Enable:    plan.RemediationText.Enable.ValueBool(),
			Text:      plan.RemediationText.Text.ValueString(),
		}
	}

	clearedButFailed, err := alert_common.ReplaceFrameworks(state.Frameworks, plan.Frameworks, updateReq,
		func(request *api_client.CustomAlert) { request.ComplianceFrameworks = nil },
		func(request api_client.CustomAlert) error {
			_, err := r.apiClient.UpdateCustomSonarAlertRule(plan.ID.ValueString(), request)
			return err
		},
	)
	if err != nil {
		alert_common.ReportFrameworkWriteFailure(ctx, resp, &plan, &plan.Frameworks, clearedButFailed, err)
		return
	}

	// Remediation text is written once, after the links are settled. Folding it
	// into the rule write would run it twice during a replace and would report a
	// remediation failure as links that were never written.
	if err := r.apiClient.UpdateCustomSonarAlertRemediation(updateReq); err != nil {
		resp.Diagnostics.AddError("Error updating Alert", err.Error())
		return
	}

	plan.Enabled = types.BoolValue(enabled)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customSonarAlertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteCustomSonarAlert(state.ID.ValueString())
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
