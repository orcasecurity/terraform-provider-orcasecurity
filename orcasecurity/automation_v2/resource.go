package automation_v2

import (
	"context"
	"encoding/json"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                     = &automationV2Resource{}
	_ resource.ResourceWithConfigure        = &automationV2Resource{}
	_ resource.ResourceWithImportState      = &automationV2Resource{}
	_ resource.ResourceWithConfigValidators = &automationV2Resource{}
)

type automationV2Resource struct {
	apiClient *api_client.APIClient
}

type automationV2FilterModel struct {
	SonarQuery types.String `tfsdk:"sonar_query"` // JSON string containing entire sonar_query
}

type automationV2AlertDismissalTemplateModel struct {
	Reason        types.String `tfsdk:"reason"`
	Justification types.String `tfsdk:"justification"`
}

type automationV2AlertScoreDecreaseTemplateModel struct {
	Reason        types.String `tfsdk:"reason"`
	Justification types.String `tfsdk:"justification"`
}

type automationV2AlertScoreIncreaseTemplateModel struct {
	Reason        types.String `tfsdk:"reason"`
	Justification types.String `tfsdk:"justification"`
}

type automationV2AlertScoreSpecifyTemplateModel struct {
	NewScore      types.Float64 `tfsdk:"new_score"`
	Reason        types.String  `tfsdk:"reason"`
	Justification types.String  `tfsdk:"justification"`
}

type automationV2ExternalConfigTemplateModel struct {
	ExternalConfigID types.String `tfsdk:"external_config_id"`
}

type automationV2ExternalConfigWithParentTemplateModel struct {
	ExternalConfigID types.String `tfsdk:"external_config_id"`
	ParentIssueID    types.String `tfsdk:"parent_issue"`
}

type automationV2EmailTemplateModel struct {
	EmailAddresses types.List `tfsdk:"email"`
	MultiAlerts    types.Bool `tfsdk:"multi_alerts"`
}

type automationV2DatadogTemplateModel struct {
	ExternalConfigID types.String `tfsdk:"external_config_id"`
	Type             types.String `tfsdk:"type"`
}

type automationV2SnoozeTemplateModel struct {
	Days          types.Int64  `tfsdk:"days"`
	Reason        types.String `tfsdk:"reason"`
	Justification types.String `tfsdk:"justification"`
}

type automationV2ResourceModel struct {
	ID            types.String             `tfsdk:"id"`
	Name          types.String             `tfsdk:"name"`
	BusinessUnits types.List               `tfsdk:"business_units"`
	Description   types.String             `tfsdk:"description"`
	Status        types.String             `tfsdk:"status"`
	Filter        *automationV2FilterModel `tfsdk:"filter"`
	EndTime       types.String             `tfsdk:"end_time"`

	AlertDismissalTemplate     *automationV2AlertDismissalTemplateModel     `tfsdk:"alert_dismissal_details"`
	AlertScoreIncreaseTemplate *automationV2AlertScoreIncreaseTemplateModel `tfsdk:"alert_score_increase_details"`
	SnoozeTemplate             *automationV2SnoozeTemplateModel             `tfsdk:"snooze_template"`
	AlertScoreDecreaseTemplate *automationV2AlertScoreDecreaseTemplateModel `tfsdk:"alert_score_decrease_details"`
	AlertScoreSpecifyTemplate  *automationV2AlertScoreSpecifyTemplateModel  `tfsdk:"alert_score_specify_details"`

	SlackTemplate *automationV2ExternalConfigTemplateModel `tfsdk:"slack_template"`

	EmailTemplate *automationV2EmailTemplateModel `tfsdk:"email_template"`

	SumoLogicTemplate     *automationV2ExternalConfigTemplateModel `tfsdk:"sumo_logic_template"`
	AzureSentinelTemplate *automationV2ExternalConfigTemplateModel `tfsdk:"azure_sentinel_template"`

	JiraCloudTemplate   *automationV2ExternalConfigWithParentTemplateModel `tfsdk:"jira_cloud_template"`
	JiraServerTemplate  *automationV2ExternalConfigWithParentTemplateModel `tfsdk:"jira_server_template"`
	AzureDevopsTemplate *automationV2ExternalConfigWithParentTemplateModel `tfsdk:"azure_devops_template"`

	PagerDutyTemplate             *automationV2ExternalConfigTemplateModel `tfsdk:"pager_duty_template"`
	OpsgenieTemplate              *automationV2ExternalConfigTemplateModel `tfsdk:"opsgenie_template"`
	MsTeamsTemplate               *automationV2ExternalConfigTemplateModel `tfsdk:"ms_teams_template"`
	SplunkTemplate                *automationV2ExternalConfigTemplateModel `tfsdk:"splunk_template"`
	AwsSecurityHubTemplate        *automationV2ExternalConfigTemplateModel `tfsdk:"aws_security_hub_template"`
	ChronicleTemplate             *automationV2ExternalConfigTemplateModel `tfsdk:"chronicle_template"`
	ServiceNowIncidentsTemplate   *automationV2ExternalConfigTemplateModel `tfsdk:"servicenow_incidents_template"`
	ServiceNowSIIncidentsTemplate *automationV2ExternalConfigTemplateModel `tfsdk:"servicenow_si_incidents_template"`
	MondayTemplate                *automationV2ExternalConfigTemplateModel `tfsdk:"monday_template"`
	LinearTemplate                *automationV2ExternalConfigTemplateModel `tfsdk:"linear_template"`
	GcpPubSubTemplate             *automationV2ExternalConfigTemplateModel `tfsdk:"gcp_pub_sub_template"`
	AwsSqsTemplate                *automationV2ExternalConfigTemplateModel `tfsdk:"aws_sqs_template"`
	AwsSnsTemplate                *automationV2ExternalConfigTemplateModel `tfsdk:"aws_sns_template"`
	AwsSecurityLakeTemplate       *automationV2ExternalConfigTemplateModel `tfsdk:"aws_security_lake_template"`
	SnowflakeTemplate             *automationV2ExternalConfigTemplateModel `tfsdk:"snowflake_template"`
	CoralogixTemplate             *automationV2ExternalConfigTemplateModel `tfsdk:"coralogix_template"`
	DatadogTemplate               *automationV2DatadogTemplateModel        `tfsdk:"datadog_template"`
	CriblTemplate                 *automationV2ExternalConfigTemplateModel `tfsdk:"cribl_template"`
	WebhookTemplate               *automationV2ExternalConfigTemplateModel `tfsdk:"webhook_template"`
	TinesTemplate                 *automationV2ExternalConfigTemplateModel `tfsdk:"tines_template"`
	TorqTemplate                  *automationV2ExternalConfigTemplateModel `tfsdk:"torq_template"`
	OpusTemplate                  *automationV2ExternalConfigTemplateModel `tfsdk:"opus_template"`
	PantherTemplate               *automationV2ExternalConfigTemplateModel `tfsdk:"panther_template"`

	OrganizationID types.String `tfsdk:"organization_id"`
}

func NewAutomationV2Resource() resource.Resource {
	return &automationV2Resource{}
}

func (r *automationV2Resource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_automation_v2"
}

func (r *automationV2Resource) Configure(_ context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *automationV2Resource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("alert_score_decrease_details"),
			path.MatchRoot("alert_score_increase_details"),
			path.MatchRoot("alert_score_specify_details"),
			path.MatchRoot("alert_dismissal_details"),
			path.MatchRoot("snooze_template"),
			path.MatchRoot("slack_template"),
			path.MatchRoot("pager_duty_template"),
			path.MatchRoot("opsgenie_template"),
			path.MatchRoot("email_template"),
			path.MatchRoot("ms_teams_template"),
			path.MatchRoot("sumo_logic_template"),
			path.MatchRoot("azure_sentinel_template"),
			path.MatchRoot("splunk_template"),
			path.MatchRoot("aws_security_hub_template"),
			path.MatchRoot("chronicle_template"),
			path.MatchRoot("jira_cloud_template"),
			path.MatchRoot("jira_server_template"),
			path.MatchRoot("servicenow_incidents_template"),
			path.MatchRoot("servicenow_si_incidents_template"),
			path.MatchRoot("monday_template"),
			path.MatchRoot("linear_template"),
			path.MatchRoot("gcp_pub_sub_template"),
			path.MatchRoot("aws_sqs_template"),
			path.MatchRoot("aws_sns_template"),
			path.MatchRoot("aws_security_lake_template"),
			path.MatchRoot("azure_devops_template"),
			path.MatchRoot("snowflake_template"),
			path.MatchRoot("coralogix_template"),
			path.MatchRoot("datadog_template"),
			path.MatchRoot("cribl_template"),
			path.MatchRoot("webhook_template"),
			path.MatchRoot("tines_template"),
			path.MatchRoot("torq_template"),
			path.MatchRoot("opus_template"),
		),
	}
}

func (r *automationV2Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func createExternalConfigTemplateSchema(serviceLabel string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: fmt.Sprintf("%s template to use for the automation.", serviceLabel),
		Attributes: map[string]schema.Attribute{
			"external_config_id": schema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("%s external service config UUID.", serviceLabel),
			},
		},
	}
}

func createExternalConfigWithParentTemplateSchema(serviceLabel, parentDescription string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: fmt.Sprintf("%s template to use for the automation.", serviceLabel),
		Attributes: map[string]schema.Attribute{
			"external_config_id": schema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("%s external service config UUID.", serviceLabel),
			},
			"parent_issue": schema.StringAttribute{
				Optional:    true,
				Description: parentDescription,
			},
		},
	}
}

func createDatadogTemplateSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Datadog template to use for the automation.",
		Attributes: map[string]schema.Attribute{
			"external_config_id": schema.StringAttribute{
				Required:    true,
				Description: "Datadog external service config UUID.",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Type of Datadog integration. Valid values: 'LOGS', 'EVENT'.",
				Validators: []validator.String{
					stringvalidator.OneOf("LOGS", "EVENT"),
				},
			},
		},
	}
}

func (r *automationV2Resource) Schema(_ context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
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
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Automation description.",
				Optional:    true,
			},
			"status": schema.StringAttribute{
				Description: "Automation status. Valid values: 'enabled', 'disabled'.",
				Optional:    true,
				Computed:    true,
			},
			"end_time": schema.StringAttribute{
				Description: "End time for the automation (ISO 8601 format). If specified, the automation will automatically disable after this time.",
				Optional:    true,
			},
			"filter": schema.SingleNestedAttribute{
				Description: "The filter that selects the alerts this automation applies to using sonar_query.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"sonar_query": schema.StringAttribute{
						Description: "Complete sonar query as JSON string. Copy the entire sonar_query structure from Orca API examples. Supports models, type, with clauses, field conditions, logical operations (and/or), and nested object queries.",
						Required:    true,
					},
				},
			},
			"alert_dismissal_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Details regarding dismissed alerts.",
				Attributes: map[string]schema.Attribute{
					"reason": schema.StringAttribute{
						Optional:    true,
						Description: "The reason these alerts are being dismissed.",
					},
					"justification": schema.StringAttribute{
						Optional:    true,
						Description: "More detailed reasoning as to why these alerts are being dismissed.",
					},
				},
			},
			"alert_score_decrease_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Details regarding decreasing the score for the selected alerts.",
				Attributes: map[string]schema.Attribute{
					"reason": schema.StringAttribute{
						Optional:    true,
						Description: "The reason these alerts are having their score decreased.",
					},
					"justification": schema.StringAttribute{
						Optional:    true,
						Description: "More detailed reasoning as to why these alerts are having their score changed.",
					},
				},
			},
			"alert_score_increase_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Details regarding increasing the score for the selected alerts.",
				Attributes: map[string]schema.Attribute{
					"reason": schema.StringAttribute{
						Optional:    true,
						Description: "The reason these alerts are having their score increased.",
					},
					"justification": schema.StringAttribute{
						Optional:    true,
						Description: "More detailed reasoning as to why these alerts are having their score changed.",
					},
				},
			},
			"alert_score_specify_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Details regarding specifying a new score for the selected alerts.",
				Attributes: map[string]schema.Attribute{
					"new_score": schema.Float64Attribute{
						Required:    true,
						Description: "New score to be assigned to the selected alerts.",
					},
					"reason": schema.StringAttribute{
						Optional:    true,
						Description: "The reason these alerts are having their score changed.",
					},
					"justification": schema.StringAttribute{
						Optional:    true,
						Description: "More detailed reasoning as to why these alerts are having their score changed.",
					},
				},
			},
			"snooze_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Snooze alert settings.",
				Attributes: map[string]schema.Attribute{
					"days": schema.Int64Attribute{
						Required:    true,
						Description: "Number of days to snooze (1-365).",
						Validators: []validator.Int64{
							int64validator.Between(1, 365),
						},
					},
					"reason": schema.StringAttribute{
						Optional:    true,
						Description: "Reason for snoozing.",
					},
					"justification": schema.StringAttribute{
						Optional:    true,
						Description: "Justification for snoozing.",
					},
				},
			},

			"aws_security_hub_template":        createExternalConfigTemplateSchema("AWS Security Hub"),
			"aws_security_lake_template":       createExternalConfigTemplateSchema("AWS Security Lake"),
			"aws_sqs_template":                 createExternalConfigTemplateSchema("AWS SQS"),
			"coralogix_template":               createExternalConfigTemplateSchema("Coralogix"),
			"gcp_pub_sub_template":             createExternalConfigTemplateSchema("GCP Pub/Sub"),
			"pager_duty_template":              createExternalConfigTemplateSchema("PagerDuty"),
			"opsgenie_template":                createExternalConfigTemplateSchema("Opsgenie"),
			"snowflake_template":               createExternalConfigTemplateSchema("Snowflake"),
			"splunk_template":                  createExternalConfigTemplateSchema("Splunk"),
			"tines_template":                   createExternalConfigTemplateSchema("Tines"),
			"torq_template":                    createExternalConfigTemplateSchema("Torq"),
			"webhook_template":                 createExternalConfigTemplateSchema("Webhook"),
			"ms_teams_template":                createExternalConfigTemplateSchema("Microsoft Teams"),
			"chronicle_template":               createExternalConfigTemplateSchema("Google Chronicle"),
			"servicenow_incidents_template":    createExternalConfigTemplateSchema("ServiceNow Incidents"),
			"servicenow_si_incidents_template": createExternalConfigTemplateSchema("ServiceNow Security Incidents"),
			"monday_template":                  createExternalConfigTemplateSchema("Monday.com"),
			"linear_template":                  createExternalConfigTemplateSchema("Linear"),
			"aws_sns_template":                 createExternalConfigTemplateSchema("AWS SNS"),
			"datadog_template":                 createDatadogTemplateSchema(),
			"cribl_template":                   createExternalConfigTemplateSchema("Cribl"),
			"opus_template":                    createExternalConfigTemplateSchema("Opus"),
			"panther_template":                 createExternalConfigTemplateSchema("Panther"),

			"sumo_logic_template":     createExternalConfigTemplateSchema("Sumo Logic"),
			"azure_sentinel_template": createExternalConfigTemplateSchema("Azure Sentinel"),

			"azure_devops_template": createExternalConfigWithParentTemplateSchema("Azure DevOps", "Automatically nest under parent issue."),
			"jira_cloud_template":   createExternalConfigWithParentTemplateSchema("Jira Cloud", "Automatically nest under this parent issue."),
			"jira_server_template":  createExternalConfigWithParentTemplateSchema("Jira Server", "Automatically nest under this parent issue."),

			"slack_template": createExternalConfigTemplateSchema("Slack"),
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
		},
	}
}

func buildV2Filter(plan *automationV2FilterModel) (api_client.AutomationV2Filter, error) {
	if plan == nil || plan.SonarQuery.IsNull() || plan.SonarQuery.IsUnknown() {
		return api_client.AutomationV2Filter{}, fmt.Errorf("filter is required")
	}

	sonarQueryJSON := plan.SonarQuery.ValueString()

	// Directly unmarshal the JSON into the struct - much cleaner
	var sonarQuery api_client.AutomationV2SonarQuery
	if err := json.Unmarshal([]byte(sonarQueryJSON), &sonarQuery); err != nil {
		return api_client.AutomationV2Filter{}, fmt.Errorf("invalid sonar_query JSON: %v", err)
	}

	return api_client.AutomationV2Filter{
		SonarQuery: sonarQuery,
	}, nil
}

func generateV2Actions(plan *automationV2ResourceModel, apiClient *api_client.APIClient) ([]api_client.AutomationV2Action, error) {
	var actions []api_client.AutomationV2Action

	if plan.SnoozeTemplate != nil {
		payload := make(map[string]interface{})
		payload["days"] = plan.SnoozeTemplate.Days.ValueInt64()
		if !plan.SnoozeTemplate.Reason.IsNull() && !plan.SnoozeTemplate.Reason.IsUnknown() {
			payload["reason"] = plan.SnoozeTemplate.Reason.ValueString()
		}
		if !plan.SnoozeTemplate.Justification.IsNull() && !plan.SnoozeTemplate.Justification.IsUnknown() {
			payload["justification"] = plan.SnoozeTemplate.Justification.ValueString()
		}
		actions = append(actions, api_client.AutomationV2Action{
			Type: api_client.AutomationSnoozeID,
			Data: payload,
		})
	}

	if plan.AlertDismissalTemplate != nil {
		payload := make(map[string]interface{})
		if !plan.AlertDismissalTemplate.Reason.IsNull() && !plan.AlertDismissalTemplate.Reason.IsUnknown() {
			payload["reason"] = plan.AlertDismissalTemplate.Reason.ValueString()
		}
		if !plan.AlertDismissalTemplate.Justification.IsNull() && !plan.AlertDismissalTemplate.Justification.IsUnknown() {
			payload["justification"] = plan.AlertDismissalTemplate.Justification.ValueString()
		}
		actions = append(actions, api_client.AutomationV2Action{
			Type: api_client.AutomationAlertDismissalID,
			Data: payload,
		})
	}

	if plan.AlertScoreDecreaseTemplate != nil {
		payload := make(map[string]interface{})
		payload["decrease_orca_score"] = 1
		if !plan.AlertScoreDecreaseTemplate.Reason.IsNull() && !plan.AlertScoreDecreaseTemplate.Reason.IsUnknown() {
			payload["reason"] = plan.AlertScoreDecreaseTemplate.Reason.ValueString()
		}
		if !plan.AlertScoreDecreaseTemplate.Justification.IsNull() && !plan.AlertScoreDecreaseTemplate.Justification.IsUnknown() {
			payload["justification"] = plan.AlertScoreDecreaseTemplate.Justification.ValueString()
		}
		actions = append(actions, api_client.AutomationV2Action{
			Type: api_client.AutomationAlertScoreChangeID,
			Data: payload,
		})
	}

	if plan.AlertScoreIncreaseTemplate != nil {
		payload := make(map[string]interface{})
		payload["increase_orca_score"] = 1
		if !plan.AlertScoreIncreaseTemplate.Reason.IsNull() && !plan.AlertScoreIncreaseTemplate.Reason.IsUnknown() {
			payload["reason"] = plan.AlertScoreIncreaseTemplate.Reason.ValueString()
		}
		if !plan.AlertScoreIncreaseTemplate.Justification.IsNull() && !plan.AlertScoreIncreaseTemplate.Justification.IsUnknown() {
			payload["justification"] = plan.AlertScoreIncreaseTemplate.Justification.ValueString()
		}
		actions = append(actions, api_client.AutomationV2Action{
			Type: api_client.AutomationAlertScoreChangeID,
			Data: payload,
		})
	}

	if plan.AlertScoreSpecifyTemplate != nil {
		payload := make(map[string]interface{})
		payload["change_orca_score"] = plan.AlertScoreSpecifyTemplate.NewScore.ValueFloat64()
		if !plan.AlertScoreSpecifyTemplate.Reason.IsNull() && !plan.AlertScoreSpecifyTemplate.Reason.IsUnknown() {
			payload["reason"] = plan.AlertScoreSpecifyTemplate.Reason.ValueString()
		}
		if !plan.AlertScoreSpecifyTemplate.Justification.IsNull() && !plan.AlertScoreSpecifyTemplate.Justification.IsUnknown() {
			payload["justification"] = plan.AlertScoreSpecifyTemplate.Justification.ValueString()
		}
		actions = append(actions, api_client.AutomationV2Action{
			Type: api_client.AutomationAlertScoreChangeID,
			Data: payload,
		})
	}

	if plan.AwsSecurityHubTemplate != nil {
		externalConfigID := plan.AwsSecurityHubTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationAWSSecurityHubID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.AwsSecurityLakeTemplate != nil {
		externalConfigID := plan.AwsSecurityLakeTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationAwsSecurityLakeID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.AwsSqsTemplate != nil {
		externalConfigID := plan.AwsSqsTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationAwsSqsID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.AzureDevopsTemplate != nil && !plan.AzureDevopsTemplate.ExternalConfigID.IsNull() {
		externalConfigID := plan.AzureDevopsTemplate.ExternalConfigID.ValueString()

		payload := make(map[string]interface{})
		if !plan.AzureDevopsTemplate.ParentIssueID.IsNull() {
			payload["parent_id"] = plan.AzureDevopsTemplate.ParentIssueID.ValueString()
		}

		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationAzureDevopsID,
			Data:           payload,
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.AzureSentinelTemplate != nil {
		externalConfigID := plan.AzureSentinelTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationAzureSentinelID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.CoralogixTemplate != nil {
		externalConfigID := plan.CoralogixTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationCoralogixID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.GcpPubSubTemplate != nil {
		externalConfigID := plan.GcpPubSubTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationGcpPubSubID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.JiraCloudTemplate != nil && !plan.JiraCloudTemplate.ExternalConfigID.IsNull() {
		externalConfigID := plan.JiraCloudTemplate.ExternalConfigID.ValueString()

		payload := make(map[string]interface{})
		if !plan.JiraCloudTemplate.ParentIssueID.IsNull() {
			payload["parent_id"] = plan.JiraCloudTemplate.ParentIssueID.ValueString()
		}

		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationJiraID,
			Data:           payload,
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.JiraServerTemplate != nil && !plan.JiraServerTemplate.ExternalConfigID.IsNull() {
		externalConfigID := plan.JiraServerTemplate.ExternalConfigID.ValueString()

		payload := make(map[string]interface{})
		if !plan.JiraServerTemplate.ParentIssueID.IsNull() {
			payload["parent_id"] = plan.JiraServerTemplate.ParentIssueID.ValueString()
		}

		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationJiraServerID,
			Data:           payload,
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.OpsgenieTemplate != nil {
		externalConfigID := plan.OpsgenieTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationOpsgenieID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.PagerDutyTemplate != nil {
		externalConfigID := plan.PagerDutyTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationPagerDutyID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.SnowflakeTemplate != nil {
		externalConfigID := plan.SnowflakeTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationSnowflakeID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.SplunkTemplate != nil {
		externalConfigID := plan.SplunkTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationSplunkID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.SumoLogicTemplate != nil {
		externalConfigID := plan.SumoLogicTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationSumoLogicID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.TinesTemplate != nil {
		externalConfigID := plan.TinesTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationTinesID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.TorqTemplate != nil {
		externalConfigID := plan.TorqTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationTorqID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.WebhookTemplate != nil {
		externalConfigID := plan.WebhookTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationWebhookID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.MsTeamsTemplate != nil {
		externalConfigID := plan.MsTeamsTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationMsTeamsID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.ChronicleTemplate != nil {
		externalConfigID := plan.ChronicleTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationChronicleID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.PantherTemplate != nil {
		externalConfigID := plan.PantherTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationPantherID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.ServiceNowIncidentsTemplate != nil {
		externalConfigID := plan.ServiceNowIncidentsTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationServiceNowIncidentsID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.ServiceNowSIIncidentsTemplate != nil {
		externalConfigID := plan.ServiceNowSIIncidentsTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationServiceNowSIIncidentsID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.MondayTemplate != nil {
		externalConfigID := plan.MondayTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationMondayID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.LinearTemplate != nil {
		externalConfigID := plan.LinearTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationLinearID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.AwsSnsTemplate != nil {
		externalConfigID := plan.AwsSnsTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationAwsSnsID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.DatadogTemplate != nil {
		externalConfigID := plan.DatadogTemplate.ExternalConfigID.ValueString()
		payload := make(map[string]interface{})
		payload["type"] = plan.DatadogTemplate.Type.ValueString()

		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationDatadogID,
			Data:           payload,
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.CriblTemplate != nil {
		externalConfigID := plan.CriblTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationCriblID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.OpusTemplate != nil {
		externalConfigID := plan.OpusTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationOpusID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	if plan.EmailTemplate != nil {
		var emailAddresses []string
		_ = plan.EmailTemplate.EmailAddresses.ElementsAs(context.Background(), &emailAddresses, false)

		payload := make(map[string]interface{})
		payload["email"] = emailAddresses
		payload["multi_alerts"] = plan.EmailTemplate.MultiAlerts.ValueBool()

		actions = append(actions, api_client.AutomationV2Action{
			Type: api_client.AutomationEmailID,
			Data: payload,
		})
	}

	if plan.SlackTemplate != nil {
		externalConfigID := plan.SlackTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationSlackID,
			Data:           make(map[string]interface{}),
			ExternalConfig: &externalConfigID,
		})
	}

	return actions, nil
}

func (r *automationV2Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan automationV2ResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	filter, err := buildV2Filter(plan.Filter)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error building Automation V2 filter",
			"Could not build Automation V2 filter: "+err.Error(),
		)
		return
	}

	actions, err := generateV2Actions(&plan, r.apiClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error generating Automation V2 actions",
			"Could not generate Automation V2 actions, unexpected error: "+err.Error(),
		)
		return
	}

	businessUnits := []string{}
	if !plan.BusinessUnits.IsNull() && !plan.BusinessUnits.IsUnknown() {
		_ = plan.BusinessUnits.ElementsAs(context.Background(), &businessUnits, false)
	}

	status := "enabled"
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		status = plan.Status.ValueString()
	}

	createReq := api_client.AutomationV2{
		Name:          plan.Name.ValueString(),
		BusinessUnits: businessUnits,
		Description:   plan.Description.ValueString(),
		Status:        status,
		Filter:        filter,
		Actions:       actions,
	}

	if !plan.EndTime.IsNull() && !plan.EndTime.IsUnknown() {
		createReq.EndTime = plan.EndTime.ValueString()
	}

	instance, err := r.apiClient.CreateAutomationV2(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Automation V2",
			"Could not create Automation V2, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.OrganizationID = types.StringValue(instance.OrganizationID)

	normalizedStatus := instance.Status
	if normalizedStatus == "success" {
		normalizedStatus = "enabled"
	}
	plan.Status = types.StringValue(normalizedStatus)

	if instance.EndTime != "" {
		plan.EndTime = types.StringValue(instance.EndTime)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *automationV2Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state automationV2ResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.apiClient.DoesAutomationV2Exist(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Automation V2",
			fmt.Sprintf("Could not read Automation V2 ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}
	if !exists {
		tflog.Warn(ctx, fmt.Sprintf("Automation V2 %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	instance, err := r.apiClient.GetAutomationV2(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Automation V2",
			fmt.Sprintf("Could not read Automation V2 ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	state.ID = types.StringValue(instance.ID)
	state.Name = types.StringValue(instance.Name)
	state.Description = types.StringValue(instance.Description)
	state.OrganizationID = types.StringValue(instance.OrganizationID)

	normalizedStatus := instance.Status
	if normalizedStatus == "success" {
		normalizedStatus = "enabled"
	}
	state.Status = types.StringValue(normalizedStatus)

	if instance.EndTime != "" {
		state.EndTime = types.StringValue(instance.EndTime)
	} else {
		state.EndTime = types.StringNull()
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *automationV2Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan automationV2ResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"ID is null",
			"Could not update Automation V2, unexpected error: "+plan.ID.ValueString(),
		)
		return
	}

	filter, err := buildV2Filter(plan.Filter)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error building Automation V2 filter",
			"Could not build Automation V2 filter: "+err.Error(),
		)
		return
	}

	actions, err := generateV2Actions(&plan, r.apiClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error generating Automation V2 actions",
			"Could not generate Automation V2 actions, unexpected error: "+err.Error(),
		)
		return
	}

	businessUnits := []string{}
	if !plan.BusinessUnits.IsNull() && !plan.BusinessUnits.IsUnknown() {
		_ = plan.BusinessUnits.ElementsAs(context.Background(), &businessUnits, false)
	}

	status := "enabled"
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		status = plan.Status.ValueString()
	}

	updateReq := api_client.AutomationV2{
		Name:          plan.Name.ValueString(),
		BusinessUnits: businessUnits,
		Description:   plan.Description.ValueString(),
		Status:        status,
		Filter:        filter,
		Actions:       actions,
	}

	if !plan.EndTime.IsNull() && !plan.EndTime.IsUnknown() {
		updateReq.EndTime = plan.EndTime.ValueString()
	}

	_, err = r.apiClient.UpdateAutomationV2(plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Automation V2",
			fmt.Sprintf("Could not update Automation V2, unexpected error: %s", err.Error()),
		)
		return
	}

	_, err = r.apiClient.GetAutomationV2(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Automation V2",
			"Could not read Automation V2 ID: "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *automationV2Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state automationV2ResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteAutomationV2(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Automation V2",
			"Could not delete Automation V2, unexpected error: "+err.Error(),
		)
		return
	}
}
