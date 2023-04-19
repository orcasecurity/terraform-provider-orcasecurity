package alerts

import (
	"context"
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
	_ resource.Resource                     = &customAlertResource{}
	_ resource.ResourceWithConfigure        = &customAlertResource{}
	_ resource.ResourceWithImportState      = &customAlertResource{}
	_ resource.ResourceWithConfigValidators = &customAlertResource{}
)

type customAlertResource struct {
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
	ID              types.String               `tfsdk:"id"`
	Name            types.String               `tfsdk:"name"`
	Description     types.String               `tfsdk:"description"`
	Rule            types.String               `tfsdk:"rule"`
	RuleType        types.String               `tfsdk:"rule_type"`
	OrganizationID  types.String               `tfsdk:"organization_id"`
	Category        types.String               `tfsdk:"category"`
	Score           types.Float64              `tfsdk:"score"`
	AllowAdjusting  types.Bool                 `tfsdk:"allow_adjusting"`
	Frameworks      []frameworkStateModel      `tfsdk:"compliance_frameworks"`
	RemediationText *remediationTextStateModel `tfsdk:"remediation_text"`
}

func NewCustomAlertResource() resource.Resource {
	return &customAlertResource{}
}

func (r *customAlertResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_custom_alert"
}

func (r *customAlertResource) Configure(_ context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *customAlertResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{}
}

func (r *customAlertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *customAlertResource) Schema(_ context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Provider Orca Security custom alerts resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Automation ID.",
			},
			"organization_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Automation name.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: "Automation description.",
				Optional:    true,
			},
			"rule": schema.StringAttribute{
				Description: "Rule query.",
				Required:    true,
			},
			"rule_type": schema.StringAttribute{
				Description: "Alert type.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"category": schema.StringAttribute{
				Description: "Category.",
				Required:    true,
			},
			"score": schema.Float64Attribute{
				Description: "Alert score.",
				Required:    true,
			},
			"allow_adjusting": schema.BoolAttribute{
				Description: "Allow Orca to adjust the score using asset context.",
				Optional:    true,
			},
			"remediation_text": schema.SingleNestedAttribute{
				Description: "Add custom manual remediation.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"enable": schema.BoolAttribute{
						Description: "Show on all alerts of this alert type for all users.",
						Optional:    true,
					},
					"text": schema.StringAttribute{
						Description: "Remediation description.",
						Required:    true,
					},
				},
			},
			"compliance_frameworks": schema.ListNestedAttribute{
				Description: "Attach compliance framework.",
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

func (r *customAlertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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

	createReq := api_client.CustomAlert{
		Name:                 plan.Name.ValueString(),
		Description:          plan.Description.ValueString(),
		Rule:                 plan.Rule.ValueString(),
		RuleType:             plan.RuleType.ValueString(),
		Category:             plan.Category.ValueString(),
		Score:                plan.Score.ValueFloat64(),
		ContextScore:         plan.AllowAdjusting.ValueBool(),
		ComplianceFrameworks: generateRequestFrameworks(plan.Frameworks),
	}
	if plan.RemediationText != nil {
		createReq.RemediationText = &api_client.CustomAlertRemediationText{
			AlertType: "", // available only after alert creation
			Enable:    plan.RemediationText.Enable.ValueBool(),
			Text:      plan.RemediationText.Text.ValueString(),
		}
	}

	instance, err := r.apiClient.CreateCustomAlert(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Alert",
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

func (r *customAlertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.GetCustomAlert(state.ID.ValueString())
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
	state.AllowAdjusting = types.BoolValue(instance.ContextScore)
	state.Score = types.Float64Value(instance.Score)
	state.RemediationText = &remediationTextStateModel{
		Enable: types.BoolValue(instance.RemediationText.Enable),
		Text:   types.StringValue(instance.RemediationText.Text),
	}

	var frameworks []frameworkStateModel
	for _, frameworkData := range instance.ComplianceFrameworks {
		frameworks = append(frameworks, frameworkStateModel{
			Name:     types.StringValue(frameworkData.Name),
			Section:  types.StringValue(frameworkData.Section),
			Priority: types.StringValue(frameworkData.Priority),
		})
	}
	state.Frameworks = frameworks

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customAlertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	updateReq := api_client.CustomAlert{
		Name:                 plan.Name.ValueString(),
		Description:          plan.Description.ValueString(),
		Rule:                 plan.Rule.ValueString(),
		RuleType:             plan.RuleType.ValueString(),
		Score:                plan.Score.ValueFloat64(),
		ContextScore:         plan.AllowAdjusting.ValueBool(),
		Category:             plan.Category.ValueString(),
		OrganizationID:       plan.OrganizationID.ValueString(),
		ComplianceFrameworks: generateRequestFrameworks(plan.Frameworks),
	}

	if plan.RemediationText != nil {
		updateReq.RemediationText = &api_client.CustomAlertRemediationText{
			AlertType: plan.RuleType.ValueString(),
			Enable:    plan.RemediationText.Enable.ValueBool(),
			Text:      plan.RemediationText.Text.ValueString(),
		}
	}

	_, err := r.apiClient.UpdateCustomAlert(plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Alert",
			"Could not update Alert, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customAlertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteCustomAlert(state.ID.ValueString())
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

func generateRequestFrameworks(frameworks []frameworkStateModel) []api_client.CustomAlertComplianceFramework {
	var frameworksReq []api_client.CustomAlertComplianceFramework
	for _, frameworkState := range frameworks {
		frameworksReq = append(frameworksReq, api_client.CustomAlertComplianceFramework{
			Name:     frameworkState.Name.ValueString(),
			Section:  frameworkState.Section.ValueString(),
			Priority: frameworkState.Priority.ValueString(),
		})
	}
	return frameworksReq
}
