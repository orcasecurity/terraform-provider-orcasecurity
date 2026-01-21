package automation

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

type automationQueryRuleRangeModel struct {
	Gte types.String `tfsdk:"gte"`
	Lte types.String `tfsdk:"lte"`
	Gt  types.String `tfsdk:"gt"`
	Lt  types.String `tfsdk:"lt"`
	Eq  types.String `tfsdk:"eq"`
}

type automationQueryRuleModel struct {
	Field         types.String                   `tfsdk:"field"`
	Includes      types.List                     `tfsdk:"includes"`
	Excludes      types.List                     `tfsdk:"excludes"`
	Prefix        types.List                     `tfsdk:"prefix"`
	ExcludePrefix types.List                     `tfsdk:"exclude_prefix"`
	Range         *automationQueryRuleRangeModel `tfsdk:"range"`
}

type automationQueryModel struct {
	Filter []automationQueryRuleModel `tfsdk:"filter"`
}

type automationAlertDismissalTemplateModel struct {
	Reason        types.String `tfsdk:"reason"`
	Justification types.String `tfsdk:"justification"`
}

// Common struct for alert score changes with reason and justification
type automationAlertScoreChangeBaseModel struct {
	Reason        types.String `tfsdk:"reason"`
	Justification types.String `tfsdk:"justification"`
}

type automationAlertScoreDecreaseTemplateModel = automationAlertScoreChangeBaseModel
type automationAlertScoreIncreaseTemplateModel = automationAlertScoreChangeBaseModel

type automationAlertScoreSpecifyTemplateModel struct {
	NewScore      types.Float64 `tfsdk:"new_score"`
	Reason        types.String  `tfsdk:"reason"`
	Justification types.String  `tfsdk:"justification"`
}

// Common struct for templates that only have a name
type automationTemplateBaseModel struct {
	Name types.String `tfsdk:"template"`
}

type automationAwsSecurityHubTemplateModel = automationTemplateBaseModel
type automationAwsSecurityLakeTemplateModel = automationTemplateBaseModel
type automationAwsSqsTemplateModel = automationTemplateBaseModel

// Common struct for templates with name and parent issue
type automationTemplateWithParentModel struct {
	Name          types.String `tfsdk:"template"`
	ParentIssueID types.String `tfsdk:"parent_issue"`
}

type automationAzureDevopsTemplateModel = automationTemplateWithParentModel
type automationJiraCloudTemplateModel = automationTemplateWithParentModel
type automationJiraServerTemplateModel = automationTemplateWithParentModel

type automationAzureSentinelTemplateModel struct {
}

type automationCoralogixTemplateModel = automationTemplateBaseModel

type automationEmailTemplateModel struct {
	EmailAddresses types.List `tfsdk:"email"`
	MultiAlerts    types.Bool `tfsdk:"multi_alerts"`
}

type automationGcpPubSubTemplateModel = automationTemplateBaseModel

type automationOpsgenieTemplateModel = automationTemplateBaseModel
type automationPagerDutyTemplateModel = automationTemplateBaseModel

type automationSlackTemplateModel struct {
	Channel   types.String `tfsdk:"channel"`
	Workspace types.String `tfsdk:"workspace"`
}

type automationSnowflakeTemplateModel struct {
}

type automationSplunkTemplateModel = automationTemplateBaseModel

type automationSumoLogicTemplateModel struct {
}

type automationTinesTemplateModel = automationTemplateBaseModel
type automationTorqTemplateModel = automationTemplateBaseModel
type automationWebhookTemplateModel = automationTemplateBaseModel

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
	AwsSecurityHubTemplate     *automationAwsSecurityHubTemplateModel     `tfsdk:"aws_security_hub_template"`
	AwsSecurityLakeTemplate    *automationAwsSecurityLakeTemplateModel    `tfsdk:"aws_security_lake_template"`
	AwsSqsTemplate             *automationAwsSqsTemplateModel             `tfsdk:"aws_sqs_template"`
	AzureDevopsTemplate        *automationAzureDevopsTemplateModel        `tfsdk:"azure_devops_template"`
	AzureSentinelTemplate      *automationAzureSentinelTemplateModel      `tfsdk:"azure_sentinel_template"`
	CoralogixTemplate          *automationCoralogixTemplateModel          `tfsdk:"coralogix_template"`
	EmailTemplate              *automationEmailTemplateModel              `tfsdk:"email_template"`
	GcpPubSubTemplate          *automationGcpPubSubTemplateModel          `tfsdk:"gcp_pub_sub_template"`
	JiraCloudTemplate          *automationJiraCloudTemplateModel          `tfsdk:"jira_cloud_template"`
	JiraServerTemplate         *automationJiraServerTemplateModel         `tfsdk:"jira_server_template"`
	OpsgenieTemplate           *automationOpsgenieTemplateModel           `tfsdk:"opsgenie_template"`
	PagerDutyTemplate          *automationPagerDutyTemplateModel          `tfsdk:"pager_duty_template"`
	SlackTemplate              *automationSlackTemplateModel              `tfsdk:"slack_template"`
	SnowflakeTemplate          *automationSnowflakeTemplateModel          `tfsdk:"snowflake_template"`
	SplunkTemplate             *automationSplunkTemplateModel             `tfsdk:"splunk_template"`
	SumoLogicTemplate          *automationSumoLogicTemplateModel          `tfsdk:"sumo_logic_template"`
	TinesTemplate              *automationTinesTemplateModel              `tfsdk:"tines_template"`
	TorqTemplate               *automationTorqTemplateModel               `tfsdk:"torq_template"`
	WebhookTemplate            *automationWebhookTemplateModel            `tfsdk:"webhook_template"`

	OrganizationID types.String `tfsdk:"organization_id"`

	// Read-only metadata fields
	CreatorID   types.String `tfsdk:"creator_id"`
	CreatorName types.String `tfsdk:"creator_name"`
	CreateTime  types.String `tfsdk:"create_time"`
	UpdateTime  types.String `tfsdk:"update_time"`
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
			path.MatchRoot("aws_security_hub_template"),
			path.MatchRoot("aws_security_lake_template"),
			path.MatchRoot("aws_sqs_template"),
			path.MatchRoot("azure_devops_template"),
			path.MatchRoot("azure_sentinel_template"),
			path.MatchRoot("coralogix_template"),
			path.MatchRoot("email_template"),
			path.MatchRoot("gcp_pub_sub_template"),
			path.MatchRoot("jira_cloud_template"),
			path.MatchRoot("jira_server_template"),
			path.MatchRoot("opsgenie_template"),
			path.MatchRoot("pager_duty_template"),
			path.MatchRoot("slack_template"),
			path.MatchRoot("snowflake_template"),
			path.MatchRoot("splunk_template"),
			path.MatchRoot("sumo_logic_template"),
			path.MatchRoot("tines_template"),
			path.MatchRoot("torq_template"),
			path.MatchRoot("webhook_template"),
		),
	}
}

func (r *automationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper to create computed string attribute with UseStateForUnknown
func computedStringAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Computed:      true,
		Description:   description,
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

// Helper to create empty nested attribute (for templates with no config)
func emptyNestedAttribute(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: description,
		Attributes:  map[string]schema.Attribute{},
	}
}

// Helper function to create simple template schema attributes
func simpleTemplateSchemaAttribute(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: description,
		Attributes: map[string]schema.Attribute{
			"template": schema.StringAttribute{
				Required:    true,
				Description: description + " template name.",
			},
		},
	}
}

// Helper function to create template schema with parent issue support
func templateWithParentSchemaAttribute(description, templateDesc string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: description,
		Attributes: map[string]schema.Attribute{
			"template": schema.StringAttribute{
				Required:    true,
				Description: templateDesc,
			},
			"parent_issue": schema.StringAttribute{
				Optional:    true,
				Description: "Automatically nest under parent issue.",
			},
		},
	}
}

// Helper function to create schema for alert score change details
func alertScoreChangeSchemaAttribute(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: description,
		Attributes: map[string]schema.Attribute{
			"reason": schema.StringAttribute{
				Optional:    true,
				Description: "The reason these alerts are having their score changed. Valid values are `Acceptable risk`, `Non-Actionable`, `Non-Production`, `Organization preferences`, and `Other`.",
			},
			"justification": schema.StringAttribute{
				Optional:    true,
				Description: "More detailed reasoning as to why these alerts are having their score changed.",
			},
		},
	}
}

func (r *automationResource) Schema(_ context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Provides an automation. You can read more about automations [here](https://docs.orcasecurity.io/docs/automations).",
		Attributes: map[string]schema.Attribute{
			"id":              computedStringAttribute("Automation ID."),
			"organization_id": computedStringAttribute("Organization ID."),
			"creator_id":      computedStringAttribute("ID of the user who created this automation."),
			"creator_name":    computedStringAttribute("Name of the user who created this automation."),
			"create_time":     computedStringAttribute("Timestamp when the automation was created."),
			"update_time":     computedStringAttribute("Timestamp when the automation was last updated."),
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
									Description: "Field to filter on. Valid values include (but are not limited to): " +
										"`category` (alert categories), " +
										"`asset_regions` (regions where assets reside), " +
										"`cve_list` (CVEs linked to alerts), " +
										"`state.risk_level` (alert risk levels), " +
										"`state.status` (alert statuses), " +
										"`state.orca_score` (numeric Orca score - use with range).",
									Required: true,
								},
								"includes": schema.ListAttribute{
									Description: "When `includes` is used, the automation applies to the specified field values.",
									Optional:    true,
									ElementType: types.StringType,
								},
								"excludes": schema.ListAttribute{
									Description: "When `excludes` is used, the automation applies to the negation of the specified field values.",
									Optional:    true,
									ElementType: types.StringType,
								},
								"prefix": schema.ListAttribute{
									Description: "Match values that start with any of the specified prefixes.",
									Optional:    true,
									ElementType: types.StringType,
								},
								"exclude_prefix": schema.ListAttribute{
									Description: "Exclude values that start with any of the specified prefixes.",
									Optional:    true,
									ElementType: types.StringType,
								},
								"range": schema.SingleNestedAttribute{
									Description: "Range-based filtering for numeric fields. Use for fields like `state.orca_score`.",
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"gte": schema.StringAttribute{
											Description: "Greater than or equal to (>=).",
											Optional:    true,
										},
										"lte": schema.StringAttribute{
											Description: "Less than or equal to (<=).",
											Optional:    true,
										},
										"gt": schema.StringAttribute{
											Description: "Greater than (>).",
											Optional:    true,
										},
										"lt": schema.StringAttribute{
											Description: "Less than (<).",
											Optional:    true,
										},
										"eq": schema.StringAttribute{
											Description: "Equal to (=).",
											Optional:    true,
										},
									},
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
			"alert_score_decrease_details": alertScoreChangeSchemaAttribute("Details regarding decreasing the score for the selected alerts."),
			"alert_score_increase_details": alertScoreChangeSchemaAttribute("Details regarding increasing the score for the selected alerts."),
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
			"aws_security_hub_template":  simpleTemplateSchemaAttribute("AWS Security Hub template to use for the automation."),
			"aws_security_lake_template": simpleTemplateSchemaAttribute("AWS Security Lake template to use for the automation."),
			"aws_sqs_template":           simpleTemplateSchemaAttribute("AWS SQS template to use for the automation."),
			"azure_devops_template": templateWithParentSchemaAttribute(
				"Azure DevOps template to use for the automation.",
				"An ADO work item template to use.",
			),
			"azure_sentinel_template": emptyNestedAttribute("Azure Sentinel template to use for the automation."),
			"coralogix_template":      simpleTemplateSchemaAttribute("Coralogix template to use for the automation."),
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
			"gcp_pub_sub_template": simpleTemplateSchemaAttribute("GCP Pub/Sub template to use for the automation."),
			"jira_cloud_template": templateWithParentSchemaAttribute(
				"Jira Cloud integration template to use for the automation.",
				"Name of the Jira Cloud integration template.",
			),
			"jira_server_template": templateWithParentSchemaAttribute(
				"Jira Server integration template to use for the automation.",
				"Name of the Jira Server integration template.",
			),
			"pager_duty_template": simpleTemplateSchemaAttribute("Pager Duty template to use for the automation."),
			"opsgenie_template":   simpleTemplateSchemaAttribute("Opsgenie template to use for the automation."),
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
			"snowflake_template":  emptyNestedAttribute("Snowflake template to use for the automation."),
			"splunk_template":     simpleTemplateSchemaAttribute("Splunk template to use for the automation."),
			"sumo_logic_template": emptyNestedAttribute("Sumo Logic template to use for the automation."),
			"tines_template":      simpleTemplateSchemaAttribute("Tines template to use for the automation."),
			"torq_template":       simpleTemplateSchemaAttribute("Torq template to use for the automation."),
			"webhook_template":    simpleTemplateSchemaAttribute("Webhook template to use for the automation."),
		},
	}
}

// Helper to add a simple template action
func addTemplateAction(actions *[]api_client.AutomationAction, template *automationTemplateBaseModel, actionType int32, orgID string) {
	if template != nil {
		payload := map[string]interface{}{"template": template.Name.ValueString()}
		*actions = append(*actions, api_client.AutomationAction{
			Type:           actionType,
			OrganizationID: orgID,
			Data:           payload,
		})
	}
}

// Helper to add a template action with parent issue support
func addTemplateWithParentAction(actions *[]api_client.AutomationAction, template *automationTemplateWithParentModel, actionType int32, orgID string) {
	if template != nil && !template.Name.IsNull() {
		payload := map[string]interface{}{"template": template.Name.ValueString()}
		if !template.ParentIssueID.IsNull() {
			payload["parent_id"] = template.ParentIssueID.ValueString()
		}
		*actions = append(*actions, api_client.AutomationAction{
			Type:           actionType,
			OrganizationID: orgID,
			Data:           payload,
		})
	}
}

func generateFilterRules(ctx context.Context, plan *automationQueryModel) (api_client.AutomationQuery, diag.Diagnostics) {
	var filterRules []api_client.AutomationFilter
	var finalDiags diag.Diagnostics

	for _, item := range plan.Filter {
		var includes []string
		if !item.Includes.IsNull() && !item.Includes.IsUnknown() {
			diags := item.Includes.ElementsAs(ctx, &includes, false)
			finalDiags.Append(diags...)
		}

		var excludes []string
		if !item.Excludes.IsNull() && !item.Excludes.IsUnknown() {
			diags := item.Excludes.ElementsAs(ctx, &excludes, false)
			finalDiags.Append(diags...)
		}

		var prefix []string
		if !item.Prefix.IsNull() && !item.Prefix.IsUnknown() {
			diags := item.Prefix.ElementsAs(ctx, &prefix, false)
			finalDiags.Append(diags...)
		}

		var excludePrefix []string
		if !item.ExcludePrefix.IsNull() && !item.ExcludePrefix.IsUnknown() {
			diags := item.ExcludePrefix.ElementsAs(ctx, &excludePrefix, false)
			finalDiags.Append(diags...)
		}

		var rangeFilter *api_client.AutomationRange
		if item.Range != nil {
			rangeFilter = &api_client.AutomationRange{}
			if !item.Range.Gte.IsNull() && !item.Range.Gte.IsUnknown() {
				gte := api_client.FlexibleString(item.Range.Gte.ValueString())
				rangeFilter.Gte = &gte
			}
			if !item.Range.Lte.IsNull() && !item.Range.Lte.IsUnknown() {
				lte := api_client.FlexibleString(item.Range.Lte.ValueString())
				rangeFilter.Lte = &lte
			}
			if !item.Range.Gt.IsNull() && !item.Range.Gt.IsUnknown() {
				gt := api_client.FlexibleString(item.Range.Gt.ValueString())
				rangeFilter.Gt = &gt
			}
			if !item.Range.Lt.IsNull() && !item.Range.Lt.IsUnknown() {
				lt := api_client.FlexibleString(item.Range.Lt.ValueString())
				rangeFilter.Lt = &lt
			}
			if !item.Range.Eq.IsNull() && !item.Range.Eq.IsUnknown() {
				eq := api_client.FlexibleString(item.Range.Eq.ValueString())
				rangeFilter.Eq = &eq
			}
		}

		filterRules = append(filterRules, api_client.AutomationFilter{
			Field:         item.Field.ValueString(),
			Includes:      includes,
			Excludes:      excludes,
			Prefix:        prefix,
			ExcludePrefix: excludePrefix,
			Range:         rangeFilter,
		})
	}
	return api_client.AutomationQuery{Filter: filterRules}, finalDiags
}

func generateActions(plan *automationResourceModel) []api_client.AutomationAction {
	var actions []api_client.AutomationAction
	orgID := plan.OrganizationID.ValueString()

	if plan.AlertDismissalTemplate != nil {
		payload := make(map[string]interface{})
		payload["reason"] = plan.AlertDismissalTemplate.Reason.ValueString()
		payload["justification"] = plan.AlertDismissalTemplate.Justification.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationAlertDismissalID,
			OrganizationID: orgID,
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
			OrganizationID: orgID,
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
			OrganizationID: orgID,
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
			OrganizationID: orgID,
			Data:           payload,
		})
	}

	// Use helper functions for simple templates
	addTemplateAction(&actions, plan.AwsSecurityHubTemplate, api_client.AutomationAWSSecurityHubID, orgID)
	addTemplateAction(&actions, plan.AwsSecurityLakeTemplate, api_client.AutomationAwsSecurityLakeID, orgID)
	addTemplateAction(&actions, plan.AwsSqsTemplate, api_client.AutomationAwsSqsID, orgID)
	addTemplateWithParentAction(&actions, plan.AzureDevopsTemplate, api_client.AutomationAzureDevopsID, orgID)

	if plan.AzureSentinelTemplate != nil {
		payload := make(map[string]interface{})
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationAzureSentinelID,
			OrganizationID: orgID,
			Data:           payload,
		})
	}

	addTemplateAction(&actions, plan.CoralogixTemplate, api_client.AutomationCoralogixID, orgID)

	if plan.EmailTemplate != nil {
		payload := make(map[string]interface{})
		var emailAddresses []string
		_ = plan.EmailTemplate.EmailAddresses.ElementsAs(context.Background(), &emailAddresses, false)

		payload["email"] = emailAddresses
		payload["multi_alerts"] = plan.EmailTemplate.MultiAlerts.ValueBool()

		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationEmailID,
			OrganizationID: orgID,
			Data:           payload,
		})
	}

	addTemplateAction(&actions, plan.GcpPubSubTemplate, api_client.AutomationGcpPubSubID, orgID)
	addTemplateWithParentAction(&actions, plan.JiraCloudTemplate, api_client.AutomationJiraID, orgID)
	addTemplateWithParentAction(&actions, plan.JiraServerTemplate, api_client.AutomationJiraID, orgID)
	addTemplateAction(&actions, plan.OpsgenieTemplate, api_client.AutomationOpsgenieID, orgID)
	addTemplateAction(&actions, plan.PagerDutyTemplate, api_client.AutomationPagerDutyID, orgID)

	if plan.SlackTemplate != nil {
		payload := make(map[string]interface{})
		payload["workspace"] = plan.SlackTemplate.Workspace.ValueString()
		payload["channel"] = plan.SlackTemplate.Channel.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationSlackID,
			OrganizationID: orgID,
			Data:           payload,
		})
	}

	if plan.SnowflakeTemplate != nil {
		payload := make(map[string]interface{})
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationSnowflakeID,
			OrganizationID: orgID,
			Data:           payload,
		})
	}

	addTemplateAction(&actions, plan.SplunkTemplate, api_client.AutomationSplunkID, orgID)

	if plan.SumoLogicTemplate != nil {
		payload := make(map[string]interface{})
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationSumoLogicID,
			OrganizationID: orgID,
			Data:           payload,
		})
	}

	addTemplateAction(&actions, plan.TinesTemplate, api_client.AutomationTinesID, orgID)
	addTemplateAction(&actions, plan.TorqTemplate, api_client.AutomationTorqID, orgID)
	addTemplateAction(&actions, plan.WebhookTemplate, api_client.AutomationWebhookID, orgID)

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

	// Set read-only metadata fields from create response
	plan.CreatorID = types.StringValue(instance.CreatorID)
	plan.CreatorName = types.StringValue(instance.CreatorName)
	plan.CreateTime = types.StringValue(instance.CreateTime)
	plan.UpdateTime = types.StringValue(instance.UpdateTime)

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

	// Set read-only metadata fields
	state.CreatorID = types.StringValue(instance.CreatorID)
	state.CreatorName = types.StringValue(instance.CreatorName)
	state.CreateTime = types.StringValue(instance.CreateTime)
	state.UpdateTime = types.StringValue(instance.UpdateTime)

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

		var prefixes []types.String
		for _, prefixValue := range rule.Prefix {
			prefixes = append(prefixes, types.StringValue(prefixValue))
		}
		prefixList, diags := types.ListValueFrom(ctx, types.StringType, prefixes)
		resp.Diagnostics.Append(diags...)

		var excludePrefixes []types.String
		for _, excludePrefixValue := range rule.ExcludePrefix {
			excludePrefixes = append(excludePrefixes, types.StringValue(excludePrefixValue))
		}
		excludePrefixList, diags := types.ListValueFrom(ctx, types.StringType, excludePrefixes)
		resp.Diagnostics.Append(diags...)

		var rangeModel *automationQueryRuleRangeModel
		if rule.Range != nil {
			rangeModel = &automationQueryRuleRangeModel{}
			if rule.Range.Gte != nil {
				rangeModel.Gte = types.StringValue(rule.Range.Gte.String())
			} else {
				rangeModel.Gte = types.StringNull()
			}
			if rule.Range.Lte != nil {
				rangeModel.Lte = types.StringValue(rule.Range.Lte.String())
			} else {
				rangeModel.Lte = types.StringNull()
			}
			if rule.Range.Gt != nil {
				rangeModel.Gt = types.StringValue(rule.Range.Gt.String())
			} else {
				rangeModel.Gt = types.StringNull()
			}
			if rule.Range.Lt != nil {
				rangeModel.Lt = types.StringValue(rule.Range.Lt.String())
			} else {
				rangeModel.Lt = types.StringNull()
			}
			if rule.Range.Eq != nil {
				rangeModel.Eq = types.StringValue(rule.Range.Eq.String())
			} else {
				rangeModel.Eq = types.StringNull()
			}
		}

		filterRules = append(filterRules, automationQueryRuleModel{
			Field:         types.StringValue(rule.Field),
			Includes:      includesList,
			Excludes:      excludesList,
			Prefix:        prefixList,
			ExcludePrefix: excludePrefixList,
			Range:         rangeModel,
		})
	}
	if resp.Diagnostics.HasError() {
		return
	}

	state.Query = &automationQueryModel{Filter: filterRules}

	// update actions
	for _, action := range instance.Actions {
		if action.IsJiraTemplate() {
			state.JiraCloudTemplate = &automationJiraCloudTemplateModel{
				Name: types.StringValue(action.Data["template"].(string)),
			}
			if action.Data["parent_id"] != nil {
				state.JiraCloudTemplate.ParentIssueID = types.StringValue(action.Data["parent_id"].(string))
			}
		}

		if action.IsAzureDevopsTemplate() {
			state.AzureDevopsTemplate = &automationAzureDevopsTemplateModel{
				Name: types.StringValue(action.Data["template"].(string)),
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

	// Fetch the updated resource to get fresh metadata
	instance, err := r.apiClient.GetAutomation(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Automation",
			"Could not read Automation ID: "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Refresh computed metadata fields from the API response
	plan.CreatorID = types.StringValue(instance.CreatorID)
	plan.CreatorName = types.StringValue(instance.CreatorName)
	plan.CreateTime = types.StringValue(instance.CreateTime)
	plan.UpdateTime = types.StringValue(instance.UpdateTime)

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
