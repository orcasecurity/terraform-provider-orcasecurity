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

type automationAwsSecurityHubTemplateModel struct {
	Name types.String `tfsdk:"template"`
}

type automationAwsSecurityLakeTemplateModel struct {
	Name types.String `tfsdk:"template"`
}

type automationAwsSqsTemplateModel struct {
	Name types.String `tfsdk:"template"`
}

type automationAzureDevopsTemplateModel struct {
	Name          types.String `tfsdk:"template"`
	ParentIssueID types.String `tfsdk:"parent_issue"`
}

type automationAzureSentinelTemplateModel struct {
}

type automationCoralogixTemplateModel struct {
	Name types.String `tfsdk:"template"`
}

type automationEmailTemplateModel struct {
	EmailAddresses types.List `tfsdk:"email"`
	MultiAlerts    types.Bool `tfsdk:"multi_alerts"`
}

type automationGcpPubSubTemplateModel struct {
	Name types.String `tfsdk:"template"`
}

type automationJiraCloudTemplateModel struct {
	Name          types.String `tfsdk:"template"`
	ParentIssueID types.String `tfsdk:"parent_issue"`
}

type automationJiraServerTemplateModel struct {
	Name          types.String `tfsdk:"template"`
	ParentIssueID types.String `tfsdk:"parent_issue"`
}

type automationOpsgenieTemplateModel struct {
	Name types.String `tfsdk:"template"`
}

type automationPagerDutyTemplateModel struct {
	Name types.String `tfsdk:"template"`
}

type automationSlackTemplateModel struct {
	Channel   types.String `tfsdk:"channel"`
	Workspace types.String `tfsdk:"workspace"`
}

type automationSnowflakeTemplateModel struct {
}

type automationSplunkTemplateModel struct {
	Name types.String `tfsdk:"template"`
}

type automationSumoLogicTemplateModel struct {
}

type automationTinesTemplateModel struct {
	Name types.String `tfsdk:"template"`
}

type automationTorqTemplateModel struct {
	Name types.String `tfsdk:"template"`
}

type automationWebhookTemplateModel struct {
	Name types.String `tfsdk:"template"`
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
			"aws_security_hub_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "AWS Security Hub template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"template": schema.StringAttribute{
						Required:    true,
						Description: "AWS Security Hub template name.",
					},
				},
			},
			"aws_security_lake_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "AWS Security Lake template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"template": schema.StringAttribute{
						Required:    true,
						Description: "AWS Security Lake template name.",
					},
				},
			},
			"aws_sqs_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "AWS SQS template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"template": schema.StringAttribute{
						Required:    true,
						Description: "AWS SQS template name.",
					},
				},
			},
			"azure_devops_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Azure DevOps template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"template": schema.StringAttribute{
						Required:    true,
						Description: "An ADO work item template to use.",
					},
					"parent_issue": schema.StringAttribute{
						Optional:    true,
						Description: "Automatically nest under parent issue.",
					},
				},
			},
			"azure_sentinel_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Azure Sentinel template to use for the automation.",
				Attributes:  map[string]schema.Attribute{},
			},
			"coralogix_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Coralogix template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"template": schema.StringAttribute{
						Required:    true,
						Description: "Coralogix template name.",
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
			"gcp_pub_sub_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "GCP Pub/Sub template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"template": schema.StringAttribute{
						Required:    true,
						Description: "GCP Pub/Sub template name.",
					},
				},
			},
			"jira_cloud_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Jira Cloud integration template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"template": schema.StringAttribute{
						Required:    true,
						Description: "Name of the Jira Cloud integration template.",
					},
					"parent_issue": schema.StringAttribute{
						Optional:    true,
						Description: "Automatically nest under this parent issue.",
					},
				},
			},
			"jira_server_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Jira Server integration template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"template": schema.StringAttribute{
						Required:    true,
						Description: "Name of the Jira Server integration template.",
					},
					"parent_issue": schema.StringAttribute{
						Optional:    true,
						Description: "Automatically nest under this parent issue.",
					},
				},
			},
			"pager_duty_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Pager Duty template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"template": schema.StringAttribute{
						Required:    true,
						Description: "Pager Duty template name.",
					},
				},
			},
			"opsgenie_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Opsgenie template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"template": schema.StringAttribute{
						Required:    true,
						Description: "Opsgenie template name.",
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
			"snowflake_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Snowflake template to use for the automation.",
				Attributes:  map[string]schema.Attribute{},
			},
			"splunk_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Splunk template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"template": schema.StringAttribute{
						Required:    true,
						Description: "Splunk template name.",
					},
				},
			},
			"sumo_logic_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Sumo Logic template to use for the automation.",
				Attributes:  map[string]schema.Attribute{},
			},
			"tines_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Tines template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"template": schema.StringAttribute{
						Required:    true,
						Description: "Tines template name.",
					},
				},
			},
			"torq_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Torq template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"template": schema.StringAttribute{
						Required:    true,
						Description: "Torq template name.",
					},
				},
			},
			"webhook_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Webhook template to use for the automation.",
				Attributes: map[string]schema.Attribute{
					"template": schema.StringAttribute{
						Required:    true,
						Description: "Webhook template name.",
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

	payload := make(map[string]interface{})

	if plan.AlertDismissalTemplate != nil {

		payload["reason"] = plan.AlertDismissalTemplate.Reason.ValueString()
		payload["justification"] = plan.AlertDismissalTemplate.Justification.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationAlertDismissalID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.AlertScoreDecreaseTemplate != nil {
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
		payload["change_orca_score"] = plan.AlertScoreSpecifyTemplate.NewScore.ValueFloat64()
		payload["reason"] = plan.AlertScoreSpecifyTemplate.Reason.ValueString()
		payload["justification"] = plan.AlertScoreSpecifyTemplate.Justification.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationAlertScoreChangeID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.AlertScoreSpecifyTemplate != nil {
		payload["change_orca_score"] = plan.AlertScoreSpecifyTemplate.NewScore.ValueFloat64()
		payload["reason"] = plan.AlertScoreSpecifyTemplate.Reason.ValueString()
		payload["justification"] = plan.AlertScoreSpecifyTemplate.Justification.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationAlertScoreChangeID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.AwsSecurityHubTemplate != nil {
		payload["template"] = plan.AwsSecurityHubTemplate.Name.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationAWSSecurityHubID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.AwsSecurityLakeTemplate != nil {
		payload["template"] = plan.AwsSecurityLakeTemplate.Name.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationAwsSecurityLakeID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.AwsSqsTemplate != nil {
		payload["template"] = plan.AwsSqsTemplate.Name.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationAwsSqsID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.AzureDevopsTemplate != nil && !plan.AzureDevopsTemplate.Name.IsNull() {
		payload["template"] = plan.AzureDevopsTemplate.Name.ValueString()
		if !plan.AzureDevopsTemplate.ParentIssueID.IsNull() {
			payload["parent_id"] = plan.AzureDevopsTemplate.ParentIssueID.ValueString()
		}

		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationAzureDevopsID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.AzureSentinelTemplate != nil {
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationAzureSentinelID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.CoralogixTemplate != nil {
		payload["template"] = plan.CoralogixTemplate.Name.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationCoralogixID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.EmailTemplate != nil {
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

	if plan.GcpPubSubTemplate != nil {
		payload["template"] = plan.GcpPubSubTemplate.Name.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationGcpPubSubID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.JiraCloudTemplate != nil && !plan.JiraCloudTemplate.Name.IsNull() {
		payload["template"] = plan.JiraCloudTemplate.Name.ValueString()
		if !plan.JiraCloudTemplate.ParentIssueID.IsNull() {
			payload["parent_id"] = plan.JiraCloudTemplate.ParentIssueID.ValueString()
		}

		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationJiraID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.JiraServerTemplate != nil && !plan.JiraServerTemplate.Name.IsNull() {
		payload["template"] = plan.JiraServerTemplate.Name.ValueString()
		if !plan.JiraServerTemplate.ParentIssueID.IsNull() {
			payload["parent_id"] = plan.JiraServerTemplate.ParentIssueID.ValueString()
		}

		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationJiraID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.OpsgenieTemplate != nil {
		payload["template"] = plan.OpsgenieTemplate.Name.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationOpsgenieID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.PagerDutyTemplate != nil {
		payload["template"] = plan.PagerDutyTemplate.Name.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationPagerDutyID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.SlackTemplate != nil {
		payload["workspace"] = plan.SlackTemplate.Workspace.ValueString()
		payload["channel"] = plan.SlackTemplate.Channel.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationSlackID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.SnowflakeTemplate != nil {
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationSnowflakeID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.SplunkTemplate != nil {
		payload["template"] = plan.SplunkTemplate.Name.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationSplunkID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.SumoLogicTemplate != nil {
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationSumoLogicID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.TinesTemplate != nil {
		payload["template"] = plan.TinesTemplate.Name.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationTinesID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.TorqTemplate != nil {
		payload["template"] = plan.TorqTemplate.Name.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationTorqID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.WebhookTemplate != nil {
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
