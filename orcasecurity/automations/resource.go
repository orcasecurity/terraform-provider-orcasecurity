package automations

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                     = &automationResource{}
	_ resource.ResourceWithConfigure        = &automationResource{}
	_ resource.ResourceWithImportState      = &automationResource{}
	_ resource.ResourceWithConfigValidators = &automationResource{}
)

type automationResource struct {
	apiClient *api_client.APIClient
}

type automationQueryRuleModel struct {
	Field    types.String `tfsdk:"field"`
	Includes types.List   `tfsdk:"includes"`
	Excludes types.List   `tfsdk:"excludes"`
}

type automationQueryModel struct {
	Filter []automationQueryRuleModel `tfsdk:"filter"`
}

type automationAlertDismissalTemplateModel struct {
	Reason        types.String `tfsdk:"reason"`
	Justification types.String `tfsdk:"justification"`
}

type automationAlertScoreDecreaseTemplateModel struct {
	Reason        types.String `tfsdk:"reason"`
	Justification types.String `tfsdk:"justification"`
}

type automationAlertScoreIncreaseTemplateModel struct {
	Reason        types.String `tfsdk:"reason"`
	Justification types.String `tfsdk:"justification"`
}

type automationAlertScoreSpecifyTemplateModel struct {
	NewScore      types.Float64 `tfsdk:"new_score"`
	Reason        types.String  `tfsdk:"reason"`
	Justification types.String  `tfsdk:"justification"`
}

type automationAzureDevopsTemplateModel struct {
	TemplateName  types.String `tfsdk:"template_name"`
	ParentIssueID types.String `tfsdk:"parent_issue"`
}

type automationEmailTemplateModel struct {
	EmailAddresses types.List `tfsdk:"email"`
	MultiAlerts    types.Bool `tfsdk:"multi_alerts"`
}

type automationJiraTemplateModel struct {
	TemplateName  types.String `tfsdk:"template_name"`
	ParentIssueID types.String `tfsdk:"parent_issue"`
}

type automationSlackTemplateModel struct {
	Channel   types.String `tfsdk:"channel"`
	Workspace types.String `tfsdk:"workspace"`
}

type automationSumoLogicTemplateModel struct {
}

type automationWebhookTemplateModel struct {
	Name types.String `tfsdk:"name"`
}

type automationResourceModel struct {
	ID            types.String          `tfsdk:"id"`
	Name          types.String          `tfsdk:"name"`
	BusinessUnits types.List            `tfsdk:"business_units"`
	Description   types.String          `tfsdk:"description"`
	Enabled       types.Bool            `tfsdk:"enabled"`
	Query         *automationQueryModel `tfsdk:"query"`

	AlertDismissalTemplate     *automationAlertDismissalTemplateModel     `tfsdk:"alert_dismissal_details"`
	AlertScoreDecreaseTemplate *automationAlertScoreDecreaseTemplateModel `tfsdk:"alert_score_decrease_details"`
	AlertScoreIncreaseTemplate *automationAlertScoreIncreaseTemplateModel `tfsdk:"alert_score_increase_details"`
	AlertScoreSpecifyTemplate  *automationAlertScoreSpecifyTemplateModel  `tfsdk:"alert_score_specify_details"`
	AzureDevopsTemplate        *automationAzureDevopsTemplateModel        `tfsdk:"azure_devops_template"`
	EmailTemplate              *automationEmailTemplateModel              `tfsdk:"email_template"`
	JiraTemplate               *automationJiraTemplateModel               `tfsdk:"jira_template"`
	SlackTemplate              *automationSlackTemplateModel              `tfsdk:"slack_template"`
	SumoLogicTemplate          *automationSumoLogicTemplateModel          `tfsdk:"sumo_logic_template"`
	WebhookTemplate            *automationWebhookTemplateModel            `tfsdk:"webhook_template"`

	OrganizationID types.String `tfsdk:"organization_id"`
}

func NewAutomationResource() resource.Resource {
	return &automationResource{}
}

func (r *automationResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_automation"
}

func (r *automationResource) Configure(_ context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *automationResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("alert_dismissal_details"),
			path.MatchRoot("alert_score_decrease_details"),
			path.MatchRoot("alert_score_increase_details"),
			path.MatchRoot("alert_score_specify_details"),
			path.MatchRoot("azure_devops_template"),
			path.MatchRoot("email_template"),
			path.MatchRoot("jira_template"),
			path.MatchRoot("slack_template"),
			path.MatchRoot("sumo_logic_template"),
			path.MatchRoot("webhook_template"),
		),
	}
}

func (r *automationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *automationResource) Schema(_ context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Provides an automation. You can read more about automations [here](https://docs.orcasecurity.io/docs/automations).",
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
			"business_units": schema.ListAttribute{
				Description: "Business units that this automation applies to, specified by their Orca ID. The business unit list cannot be changed after creation.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"description": schema.StringAttribute{
				Description: "Automation description.",
				Optional:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Automation status.",
				Required:    true,
			},
			"query": schema.SingleNestedAttribute{
				Description: "The query that selects the alerts this automation applies to.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"filter": schema.ListNestedAttribute{
						Description: "List of filters upon which alerts are selected.",
						Required:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"field": schema.StringAttribute{
									Description: `When ` + "`includes`" + " is used, the automation applies to the specified field. Valid values include (but are not limited to):\n" +
										"  - `category`" + " - alert categories\n" +
										"  - `asset_regions`" + " - regions where the assets reside\n" +
										"  - `cve_list`" + " - CVEs linked to the alerts\n" +
										"  - `state.risk_level`" + " - alert risk scores\n" +
										"  - `state.status`" + " - alert statuses\n",
									Required: true,
								},
								"includes": schema.ListAttribute{
									Description: "When `includes` is used, the automation applies to the specified field.",
									Optional:    true,
									ElementType: types.StringType,
								},
								"excludes": schema.ListAttribute{
									Description: "When `excludes` is used, the automation applies to the negation of the specified field.",
									Optional:    true,
									ElementType: types.StringType,
								},
							},
						},
					},
				},
			},
			"alert_dismissal_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Details regarding dismissed alerts.",
				Attributes: map[string]schema.Attribute{
					"reason": schema.StringAttribute{
						Optional:    true,
						Description: "The reason these alerts are being dismissed. Valid values are `Acceptable Risk`, `False Positives`, `Non-Actionable`, `Non-Production`, and `Other`.",
					},
					"justification": schema.StringAttribute{
						Optional:    true,
						Description: "More detailed reasoning as to why these alerts are being dismissed.",
					},
				},
			},
			"alert_score_decrease_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Details regarding the new score for the selected alerts.",
				Attributes: map[string]schema.Attribute{
					"reason": schema.StringAttribute{
						Optional:    true,
						Description: "The reason these alerts are being dismissed. Valid values are `Acceptable risk`, `Non-Actionable`, `Non-Production`, `Organization preferences`, and `Other`.",
					},
					"justification": schema.StringAttribute{
						Optional:    true,
						Description: "More detailed reasoning as to why these alerts are having their score changed.",
					},
				},
			},
			"alert_score_increase_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Details regarding the new score for the selected alerts.",
				Attributes: map[string]schema.Attribute{
					"reason": schema.StringAttribute{
						Optional:    true,
						Description: "The reason these alerts are being dismissed. Valid values are `Acceptable risk`, `Non-Actionable`, `Non-Production`, `Organization preferences`, and `Other`.",
					},
					"justification": schema.StringAttribute{
						Optional:    true,
						Description: "More detailed reasoning as to why these alerts are having their score changed.",
					},
				},
			},
			"alert_score_specify_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Details regarding the new score for the selected alerts.",
				Attributes: map[string]schema.Attribute{
					"new_score": schema.Float64Attribute{
						Required:    true,
						Description: "New score to be assigned to the selected alerts.",
					},
					"reason": schema.StringAttribute{
						Optional:    true,
						Description: "The reason these alerts are being dismissed. Valid values are `Acceptable risk`, `Non-Actionable`, `Non-Production`, `Organization preferences`, and `Other`.",
					},
					"justification": schema.StringAttribute{
						Optional:    true,
						Description: "More detailed reasoning as to why these alerts are having their score changed.",
					},
				},
			},
			"azure_devops_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Azure DevOps template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"template_name": schema.StringAttribute{
						Required:    true,
						Description: "An ADO work item template to use.",
					},
					"parent_issue": schema.StringAttribute{
						Optional:    true,
						Description: "Automatically nest under parent issue.",
					},
				},
			},
			"email_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Email settings.",
				Attributes: map[string]schema.Attribute{
					"email": schema.ListAttribute{
						ElementType: types.StringType,
						Required:    true,
						Description: "Email addresses to send the alerts to",
					},
					"multi_alerts": schema.BoolAttribute{
						Required:    true,
						Description: "`true` means multiple alerts will be aggregated into 1 email. `false` means the email recipients will receive 1 email per alert.",
					},
				},
			},
			"jira_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Jira integration template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"template_name": schema.StringAttribute{
						Required:    true,
						Description: "Name of the Jira integration template.",
					},
					"parent_issue": schema.StringAttribute{
						Optional:    true,
						Description: "Automatically nest under this parent issue.",
					},
				},
			},
			"slack_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Slack template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"workspace": schema.StringAttribute{
						Required:    true,
						Description: "Slack workspace to use.",
					},
					"channel": schema.StringAttribute{
						Required:    true,
						Description: "Slack channel ID to post the alert to. Example: `C04CLKEF7PU`.",
					},
				},
			},
			"sumo_logic_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Sumo Logic template to use for the automation.",
				Attributes:  map[string]schema.Attribute{},
			},
			"webhook_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Webhook template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:    true,
						Description: "Webhook name.",
					},
				},
			},
		},
	}
}

func generateFilterRules(ctx context.Context, plan *automationQueryModel) (api_client.AutomationQuery, diag.Diagnostics) {
	var filterRules []api_client.AutomationFilter
	var finalDiags diag.Diagnostics

	for _, item := range plan.Filter {
		var includes []string
		if !item.Includes.IsNull() {
			diags := item.Includes.ElementsAs(ctx, &includes, false)
			finalDiags.Append(diags...)
		}

		var excludes []string
		if !item.Excludes.IsNull() {
			diags := item.Excludes.ElementsAs(ctx, &excludes, false)
			finalDiags.Append(diags...)
		}

		filterRules = append(filterRules, api_client.AutomationFilter{
			Field:    item.Field.ValueString(),
			Includes: includes,
			Excludes: excludes,
		})
	}
	return api_client.AutomationQuery{Filter: filterRules}, finalDiags
}

func generateActions(plan *automationResourceModel) []api_client.AutomationAction {
	var actions []api_client.AutomationAction

	if plan.AlertDismissalTemplate != nil {
		payload := make(map[string]interface{})
		payload["reason"] = plan.AlertDismissalTemplate.Reason.ValueString()
		payload["justification"] = plan.AlertDismissalTemplate.Justification.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationAlertDismissalID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.AlertScoreDecreaseTemplate != nil {
		payload := make(map[string]interface{})
		payload["decrease_orca_score"] = 1
		payload["reason"] = plan.AlertScoreDecreaseTemplate.Reason.ValueString()
		payload["justification"] = plan.AlertScoreDecreaseTemplate.Justification.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationAlertScoreChangeID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.AlertScoreIncreaseTemplate != nil {
		payload := make(map[string]interface{})
		payload["increase_orca_score"] = 1
		payload["reason"] = plan.AlertScoreIncreaseTemplate.Reason.ValueString()
		payload["justification"] = plan.AlertScoreIncreaseTemplate.Justification.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationAlertScoreChangeID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.AlertScoreSpecifyTemplate != nil {
		payload := make(map[string]interface{})
		payload["change_orca_score"] = plan.AlertScoreSpecifyTemplate.NewScore.ValueFloat64()
		payload["reason"] = plan.AlertScoreSpecifyTemplate.Reason.ValueString()
		payload["justification"] = plan.AlertScoreSpecifyTemplate.Justification.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationAlertScoreChangeID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.JiraTemplate != nil && !plan.JiraTemplate.TemplateName.IsNull() {
		payload := make(map[string]interface{})
		payload["template"] = plan.JiraTemplate.TemplateName.ValueString()
		if !plan.JiraTemplate.ParentIssueID.IsNull() {
			payload["parent_id"] = plan.JiraTemplate.ParentIssueID.ValueString()
		}

		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationJiraID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.AzureDevopsTemplate != nil && !plan.AzureDevopsTemplate.TemplateName.IsNull() {
		payload := make(map[string]interface{})
		payload["template"] = plan.AzureDevopsTemplate.TemplateName.ValueString()
		if !plan.AzureDevopsTemplate.ParentIssueID.IsNull() {
			payload["parent_id"] = plan.AzureDevopsTemplate.ParentIssueID.ValueString()
		}

		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationAzureDevopsID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.EmailTemplate != nil {
		payload := make(map[string]interface{})

		var emailAddresses []string
		_ = plan.EmailTemplate.EmailAddresses.ElementsAs(context.Background(), &emailAddresses, false)

		payload["email"] = emailAddresses

		payload["multi_alerts"] = plan.EmailTemplate.MultiAlerts.ValueBool()

		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationEmailID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.SlackTemplate != nil {
		payload := make(map[string]interface{})
		payload["workspace"] = plan.SlackTemplate.Workspace.ValueString()
		payload["channel"] = plan.SlackTemplate.Channel.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationSlackID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.SumoLogicTemplate != nil {
		payload := make(map[string]interface{})
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationSumoLogicID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.WebhookTemplate != nil {
		payload := make(map[string]interface{})
		payload["template"] = plan.WebhookTemplate.Name.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationWebhookID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}
	return actions
}

func (r *automationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan automationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	filterQuery, filterDiags := generateFilterRules(ctx, plan.Query)
	diags.Append(filterDiags...)

	actions := generateActions(&plan)

	businessUnits := []string{}
	_ = plan.BusinessUnits.ElementsAs(context.Background(), businessUnits, false)

	createReq := api_client.Automation{
		Actions:       actions,
		Name:          plan.Name.ValueString(),
		BusinessUnits: businessUnits,
		Description:   plan.Description.ValueString(),
		Enabled:       plan.Enabled.ValueBool(),
		Query:         filterQuery,
	}
	instance, err := r.apiClient.CreateAutomation(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Automation",
			"Could not create Automation, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.OrganizationID = types.StringValue(instance.OrganizationID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *automationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state automationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.apiClient.DoesAutomationExist(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Automation",
			fmt.Sprintf("Could not read Automation ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}
	if !exists {
		tflog.Warn(ctx, fmt.Sprintf("Automation %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	instance, err := r.apiClient.GetAutomation(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Automation",
			fmt.Sprintf("Could not read Automation ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	state.ID = types.StringValue(instance.ID)
	state.Name = types.StringValue(instance.Name)
	state.Description = types.StringValue(instance.Description)
	state.OrganizationID = types.StringValue(instance.OrganizationID)

	// update query filters
	var filterRules []automationQueryRuleModel
	for _, rule := range instance.Query.Filter {
		var includes []types.String
		for _, includeValue := range rule.Includes {
			includes = append(includes, types.StringValue(includeValue))
		}
		includesList, diags := types.ListValueFrom(ctx, types.StringType, includes)
		resp.Diagnostics.Append(diags...)

		var excludes []types.String
		for _, excludeValue := range rule.Excludes {
			excludes = append(excludes, types.StringValue(excludeValue))
		}
		excludesList, diags := types.ListValueFrom(ctx, types.StringType, excludes)
		resp.Diagnostics.Append(diags...)

		filterRules = append(filterRules, automationQueryRuleModel{
			Field:    types.StringValue(rule.Field),
			Includes: includesList,
			Excludes: excludesList,
		})
	}
	if resp.Diagnostics.HasError() {
		return
	}

	state.Query = &automationQueryModel{Filter: filterRules}

	// update actions
	for _, action := range instance.Actions {
		if action.IsJiraTemplate() {
			state.JiraTemplate = &automationJiraTemplateModel{
				TemplateName: types.StringValue(action.Data["template"].(string)),
			}
			if action.Data["parent_id"] != nil {
				state.JiraTemplate.ParentIssueID = types.StringValue(action.Data["parent_id"].(string))
			}
		}

		if action.IsAzureDevopsTemplate() {
			state.AzureDevopsTemplate = &automationAzureDevopsTemplateModel{
				TemplateName: types.StringValue(action.Data["template"].(string)),
			}
			if action.Data["parent_id"] != nil {
				state.AzureDevopsTemplate.ParentIssueID = types.StringValue(action.Data["parent_id"].(string))
			}
		}

		if action.IsSumoLogicTemplate() {
			state.SumoLogicTemplate = &automationSumoLogicTemplateModel{}
		}

		if action.IsWebhookTemplate() {
			state.WebhookTemplate = &automationWebhookTemplateModel{
				Name: types.StringValue(action.Data["template"].(string)),
			}
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *automationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan automationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"ID is null",
			"Could not update Automation, unexpected error: "+plan.ID.ValueString(),
		)
		return
	}

	filterQuery, filterDiags := generateFilterRules(ctx, plan.Query)
	diags.Append(filterDiags...)

	actions := generateActions(&plan)

	businessUnits := []string{}
	_ = plan.BusinessUnits.ElementsAs(context.Background(), businessUnits, false)

	updateReq := api_client.Automation{
		Actions:        actions,
		Query:          filterQuery,
		Name:           plan.Name.ValueString(),
		BusinessUnits:  businessUnits,
		Description:    plan.Description.ValueString(),
		Enabled:        plan.Enabled.ValueBool(),
		OrganizationID: plan.OrganizationID.ValueString(),
	}

	_, err := r.apiClient.UpdateAutomation(plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Automation",
			fmt.Sprintf("Could not update Automation, unexpected error: %d::", len(businessUnits))+err.Error(),
		)
		return
	}

	_, err = r.apiClient.GetAutomation(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Automation",
			"Could not read Automation ID: "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *automationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state automationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteAutomation(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Automation",
			"Could not delete Automation, unexpected error: "+err.Error(),
		)
		return
	}
}
