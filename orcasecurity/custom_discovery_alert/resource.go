package custom_discovery_alert

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
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
	_ resource.Resource                     = &customDiscoveryAlertResource{}
	_ resource.ResourceWithConfigure        = &customDiscoveryAlertResource{}
	_ resource.ResourceWithImportState      = &customDiscoveryAlertResource{}
	_ resource.ResourceWithConfigValidators = &customDiscoveryAlertResource{}
)

type customDiscoveryAlertResource struct {
	apiClient *api_client.APIClient
}

type frameworkStateModel struct {
	Name     types.String `tfsdk:"name"`
	Section  types.String `tfsdk:"section"`
	Priority types.String `tfsdk:"priority"`
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
	Severity        types.Float64              `tfsdk:"severity"`
	Frameworks      []frameworkStateModel      `tfsdk:"compliance_frameworks"`
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
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Custom alert ID.",
			},
			"rule_type": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Custom alert rule type (unique, Orca-computed identifier).",
			},
			"organization_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Identifier your Orca organization.",
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
			"category": schema.StringAttribute{
				Description: "Category of the risk that the alert presents.",
				Required:    true,
			},
			"rule_json": schema.StringAttribute{
				Description: "The discovery query (JSON) used to define the rule.",
				Optional:    true,
			},
			"severity": schema.Float64Attribute{
				Description: "Alert severity.",
				Required:    true,
			},
			"orca_score": schema.Float64Attribute{
				Description: "The base score of the alert.",
				Required:    true,
			},
			"context_score": schema.BoolAttribute{
				Description: "Allows Orca to adjust the score using asset context.",
				Required:    true,
			},
			"remediation_text": schema.SingleNestedAttribute{
				Description: "A container for the remediation instructions that will appear on the 'Remediation' tab for the alert.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"enable": schema.BoolAttribute{
						Description: "Show on all alerts of this alert type for all users.",
						Optional:    true,
					},
					"text": schema.StringAttribute{
						Description: "The remediation instructions.",
						Required:    true,
					},
				},
			},
			"compliance_frameworks": schema.ListNestedAttribute{
				Description: "The custom compliance frameworks that this control relates to.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Framework name.",
						},
						"section": schema.StringAttribute{
							Required:    true,
							Description: "Section.",
						},
						"priority": schema.StringAttribute{
							Required:    true,
							Description: "Priority.",
						},
					},
				},
			},
		},
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

	createReq := api_client.CustomDiscoveryAlert{
		Name:                 plan.Name.ValueString(),
		RuleJson:             query,
		Description:          plan.Description.ValueString(),
		RuleType:             plan.RuleType.ValueString(),
		Category:             plan.Category.ValueString(),
		Severity:             plan.Severity.ValueFloat64(),
		OrcaScore:            plan.OrcaScore.ValueFloat64(),
		ContextScore:         plan.ContextScore.ValueBool(),
		ComplianceFrameworks: generateRequestFrameworks(plan.Frameworks),
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

	instance, err := r.apiClient.GetCustomDiscoveryAlert(state.ID.ValueString())
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
	state.RuleType = types.StringValue(instance.RuleType)
	state.Severity = types.Float64Value(instance.Severity)
	state.OrganizationID = types.StringValue(instance.OrganizationID)
	state.Category = types.StringValue(instance.Category)
	state.ContextScore = types.BoolValue(instance.ContextScore)
	state.OrcaScore = types.Float64Value(instance.OrcaScore)

	if instance.RemediationText.Text != "" {
		state.RemediationText = &remediationTextStateModel{
			Enable: types.BoolValue(instance.RemediationText.Enable),
			Text:   types.StringValue(instance.RemediationText.Text),
		}
	}

	/*var frameworks []frameworkStateModel
	for _, frameworkData := range instance.ComplianceFrameworks {
		frameworks = append(frameworks, frameworkStateModel{
			Name:     types.StringValue(frameworkData.Name),
			Section:  types.StringValue(frameworkData.Section),
			Priority: types.StringValue(frameworkData.Priority),
		})
	}
	state.Frameworks = frameworks*/

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

	updateReq := api_client.CustomDiscoveryAlert{
		Name:                 plan.Name.ValueString(),
		Description:          plan.Description.ValueString(),
		RuleJson:             query,
		Severity:             plan.Severity.ValueFloat64(),
		RuleType:             plan.RuleType.ValueString(),
		OrcaScore:            plan.OrcaScore.ValueFloat64(),
		ContextScore:         plan.ContextScore.ValueBool(),
		Category:             plan.Category.ValueString(),
		OrganizationID:       plan.OrganizationID.ValueString(),
		ComplianceFrameworks: generateRequestFrameworks(plan.Frameworks),
	}

	if plan.RemediationText != nil {
		updateReq.RemediationText = &api_client.CustomDiscoveryAlertRemediationText{
			AlertType: plan.RuleType.ValueString(),
			Enable:    plan.RemediationText.Enable.ValueBool(),
			Text:      plan.RemediationText.Text.ValueString(),
		}
	}

	_, err = r.apiClient.UpdateCustomDiscoveryAlert(plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Alert",
			"Could not update Alert, unexpected error: "+err.Error(),
		)
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

func generateRequestFrameworks(frameworks []frameworkStateModel) []api_client.CustomDiscoveryAlertComplianceFramework {
	var frameworksReq []api_client.CustomDiscoveryAlertComplianceFramework
	for _, frameworkState := range frameworks {
		frameworksReq = append(frameworksReq, api_client.CustomDiscoveryAlertComplianceFramework{
			Name:     frameworkState.Name.ValueString(),
			Section:  frameworkState.Section.ValueString(),
			Priority: frameworkState.Priority.ValueString(),
		})
	}
	return frameworksReq
}
