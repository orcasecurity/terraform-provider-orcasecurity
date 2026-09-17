package custom_sonar_alert

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/alert_common"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Custom alert ID.",
			},
			"organization_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Orca organization ID.",
			},
			"name": schema.StringAttribute{
				Description: "Custom alert name.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: "Custom alert description.",
				Optional:    true,
			},
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
			"category": schema.StringAttribute{
				Description: "Alert category. Valid values are `Access control`, `Authentication`, `Best practices`, `Data at risk`, `Data protection`, `IAM misconfigurations`, `Lateral movement`, `Logging and monitoring`, `Malicious activity`, `Malware`, `Neglected assets`, `Network misconfigurations`, `Source code vulnerabilities`, `Suspicious activity`, `System integrity`, `Vendor services misconfigurations`, `Vulnerabilities`, and `Workload misconfigurations`.",
				Required:    true,
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
			"remediation_text": schema.SingleNestedAttribute{
				Description: "A container for the remediation instructions that will appear on the 'Remediation' tab for the alert.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"enable": schema.BoolAttribute{
						Description: "Whether or not all users are able to see the remediation instructions for this alert. To enable all users to see them, set this to `true`.",
						Optional:    true,
					},
					"text": schema.StringAttribute{
						Description: "Remediation description.",
						Required:    true,
					},
				},
			},
			"compliance_frameworks": schema.ListNestedAttribute{
				Description: "The custom compliance framework(s) that this alert relates to. In the context of a compliance framework, alerts correspond to controls. Omit the attribute to leave the existing links untouched - they may be owned by the Orca UI or by a custom compliance framework resource. Set it to `[]` to detach the alert from every framework.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Custom framework name.",
						},
						"section": schema.StringAttribute{
							Required:    true,
							Description: "Custom framework section. For nested sections, join the levels with `/` (e.g. `Identify/Risk Assessment/Vulnerabilities in assets are identified`); up to three levels are supported.",
						},
						"priority": schema.StringAttribute{
							Required:    true,
							Description: "Custom framework control priority. Valid values are `high`, `medium`, and `low`.",
						},
					},
				},
			},
		},
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

	createFrameworks, frameworkDiags := generateRequestFrameworks(ctx, plan.Frameworks)
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

	instance, err = r.apiClient.GetCustomSonarAlert(instance.ID)
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

	if plan.Frameworks.IsUnknown() {
		frameworks, diags := frameworksToList(ctx, instance.ComplianceFrameworks)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Frameworks = frameworks
	}

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

	exists, err := r.apiClient.DoesCustomSonarAlertExist(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Alert",
			fmt.Sprintf("Could not read Alert ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	if !exists {
		tflog.Warn(ctx, fmt.Sprintf("Alert %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
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

	frameworks, diags := frameworksToList(ctx, instance.ComplianceFrameworks)
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

	updateFrameworks, frameworkDiags := generateRequestFrameworks(ctx, plan.Frameworks)
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

	clearedButFailed, err := alert_common.ReplaceFrameworks(alert_common.FrameworksCount(state.Frameworks) > 0, alert_common.FrameworksCount(plan.Frameworks) > 0,
		func() error {
			clearReq := updateReq
			clearReq.ComplianceFrameworks = nil
			_, err := r.apiClient.UpdateCustomSonarAlert(plan.ID.ValueString(), clearReq)
			return err
		},
		func() error {
			_, err := r.apiClient.UpdateCustomSonarAlert(plan.ID.ValueString(), updateReq)
			return err
		},
	)
	if err != nil {
		if clearedButFailed {
			// Clear succeeded remotely; persist empty frameworks on apply failure.
			plan.Frameworks = types.ListValueMust(alert_common.FrameworkObjectType(), nil)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		}
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

func generateRequestFrameworks(ctx context.Context, list types.List) ([]api_client.CustomSonarAlertComplianceFramework, diag.Diagnostics) {
	frameworks, diags := alert_common.FrameworksFromList(ctx, list)
	if diags.HasError() {
		return nil, diags
	}

	var frameworksReq []api_client.CustomSonarAlertComplianceFramework
	for _, framework := range frameworks {
		category, subCategory, subSubCategory := api_client.SplitComplianceSection(framework.Section.ValueString())
		frameworksReq = append(frameworksReq, api_client.CustomSonarAlertComplianceFramework{
			Name:           framework.Name.ValueString(),
			Category:       category,
			SubCategory:    subCategory,
			SubSubCategory: subSubCategory,
			Priority:       framework.Priority.ValueString(),
		})
	}
	return frameworksReq, diags
}

func frameworksToList(ctx context.Context, frameworks []api_client.CustomSonarAlertComplianceFramework) (types.List, diag.Diagnostics) {
	values := make([]alert_common.Framework, 0, len(frameworks))
	for _, framework := range frameworks {
		values = append(values, alert_common.Framework{
			Name: types.StringValue(framework.Name),
			Section: types.StringValue(api_client.JoinComplianceSection(
				framework.Category, framework.SubCategory, framework.SubSubCategory)),
			Priority: types.StringValue(framework.Priority),
		})
	}
	return alert_common.FrameworksToList(ctx, values)
}
