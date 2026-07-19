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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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

const scoreChangeJustificationDescription = "More detailed reasoning as to why these alerts are having their score changed. Optional; empty string is treated as omitted."

// optionalStringAttr builds a plain Optional string attribute. Empty string is
// treated as omitted at the API layer (setOptionalString drops it), so the API
// never returns it; on a normal refresh the prior state value is preserved
// untouched, so `reason = ""` round-trips cleanly. We must NOT use a plan
// modifier to normalize "" to null here: the Plugin Framework rejects any
// planned value that differs from config on an Optional (non-Computed)
// attribute ("planned value ... does not match config value").
func optionalStringAttr(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Description: description,
	}
}

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
	AssetTagKeys   types.List `tfsdk:"asset_tag_keys"`
	CustomTagKeys  types.List `tfsdk:"custom_tag_keys"`
}

type automationV2RemediationTemplateModel struct {
	RemediationAction types.String `tfsdk:"remediation_action"`
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

	RemediationTemplate *automationV2RemediationTemplateModel `tfsdk:"remediation_template"`

	SumoLogicTemplate     *automationV2ExternalConfigTemplateModel `tfsdk:"sumo_logic_template"`
	AzureSentinelTemplate *automationV2ExternalConfigTemplateModel `tfsdk:"azure_sentinel_template"`
	ApiTokenTemplate      *automationV2ExternalConfigTemplateModel `tfsdk:"api_token_template"`

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

	OrganizationID  types.String `tfsdk:"organization_id"`
	ApplyOnExisting types.Bool   `tfsdk:"apply_on_existing"`
	Priority        types.Int64  `tfsdk:"priority"`
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
			path.MatchRoot("api_token_template"),
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
			path.MatchRoot("panther_template"),
			path.MatchRoot("remediation_template"),
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
			"apply_on_existing": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "When true, retroactively applies the automation's actions to existing alerts matching the filter at creation time. Only honored on POST; changing this value forces resource replacement.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
					boolplanmodifier.RequiresReplace(),
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
			"priority": schema.Int64Attribute{
				Optional: true,
				Description: "Evaluation-order priority (1 = evaluated first). Priorities form a global, " +
					"dense 1..N ordering across all automations in the organization; the server renumbers " +
					"other automations whenever one moves. Omit to leave ordering unmanaged by Terraform " +
					"(existing configurations are unaffected). Setting it requires a token with the global " +
					"Rules Create (admin) permission. A value above the current number of automations fails " +
					"the apply and reports the actual placement.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
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
					"reason":        optionalStringAttr("The reason these alerts are being dismissed. Optional; empty string is treated as omitted."),
					"justification": optionalStringAttr("More detailed reasoning as to why these alerts are being dismissed. Optional; empty string is treated as omitted."),
				},
			},
			"alert_score_decrease_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Details regarding decreasing the score for the selected alerts.",
				Attributes: map[string]schema.Attribute{
					"reason":        optionalStringAttr("The reason these alerts are having their score decreased. Optional; empty string is treated as omitted."),
					"justification": optionalStringAttr(scoreChangeJustificationDescription),
				},
			},
			"alert_score_increase_details": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Details regarding increasing the score for the selected alerts.",
				Attributes: map[string]schema.Attribute{
					"reason":        optionalStringAttr("The reason these alerts are having their score increased. Optional; empty string is treated as omitted."),
					"justification": optionalStringAttr(scoreChangeJustificationDescription),
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
					"reason":        optionalStringAttr("The reason these alerts are having their score changed. Optional; empty string is treated as omitted."),
					"justification": optionalStringAttr(scoreChangeJustificationDescription),
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
					"reason":        optionalStringAttr("Reason for snoozing. Optional; empty string is treated as omitted."),
					"justification": optionalStringAttr("Justification for snoozing. Optional; empty string is treated as omitted."),
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
			"api_token_template":      createExternalConfigTemplateSchema("API Token"),

			"azure_devops_template": createExternalConfigWithParentTemplateSchema("Azure DevOps", "Automatically nest under parent issue."),
			"jira_cloud_template":   createExternalConfigWithParentTemplateSchema("Jira Cloud", "Automatically nest under this parent issue."),
			"jira_server_template":  createExternalConfigWithParentTemplateSchema("Jira Server", "Automatically nest under this parent issue."),

			"slack_template": createExternalConfigTemplateSchema("Slack"),
			"email_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Email settings. Provide at least one recipient mode: `email`, `asset_tag_keys`, or `custom_tag_keys`.",
				Validators: []validator.Object{
					AtLeastOneChildSet("email", "asset_tag_keys", "custom_tag_keys"),
				},
				Attributes: map[string]schema.Attribute{
					"email": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "Email addresses to send the alerts to.",
					},
					"multi_alerts": schema.BoolAttribute{
						Optional:    true,
						Description: "`true` means multiple alerts will be aggregated into 1 email. `false` means the email recipients will receive 1 email per alert.",
					},
					"asset_tag_keys": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "Asset tag keys whose values are used to derive the email recipients (\"email by tag\").",
					},
					"custom_tag_keys": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "Custom tag keys whose values are used to derive the email recipients.",
					},
				},
			},
			"remediation_template": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Remediation (Auto Remediate) settings.",
				Attributes: map[string]schema.Attribute{
					"remediation_action": schema.StringAttribute{
						Required:    true,
						Description: "The remediation action ID to run (e.g. `AWS-S3-004`).",
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

func setOptionalString(payload map[string]interface{}, key string, value types.String) {
	if value.IsNull() || value.IsUnknown() {
		return
	}
	if v := value.ValueString(); v != "" {
		payload[key] = v
	}
}

func appendExternalConfigAction(actions []api_client.AutomationV2Action, tmpl *automationV2ExternalConfigTemplateModel, actionType int32) []api_client.AutomationV2Action {
	if tmpl == nil {
		return actions
	}
	externalConfigID := tmpl.ExternalConfigID.ValueString()
	return append(actions, api_client.AutomationV2Action{
		Type:           actionType,
		Data:           map[string]interface{}{},
		ExternalConfig: &externalConfigID,
	})
}

func appendExternalConfigWithParentAction(actions []api_client.AutomationV2Action, tmpl *automationV2ExternalConfigWithParentTemplateModel, actionType int32) []api_client.AutomationV2Action {
	if tmpl == nil || tmpl.ExternalConfigID.IsNull() {
		return actions
	}
	externalConfigID := tmpl.ExternalConfigID.ValueString()
	payload := map[string]interface{}{}
	if !tmpl.ParentIssueID.IsNull() {
		payload["parent_id"] = tmpl.ParentIssueID.ValueString()
	}
	return append(actions, api_client.AutomationV2Action{
		Type:           actionType,
		Data:           payload,
		ExternalConfig: &externalConfigID,
	})
}

func appendReasonJustificationAction(actions []api_client.AutomationV2Action, actionType int32, reason, justification types.String, extra map[string]interface{}) []api_client.AutomationV2Action {
	payload := extra
	if payload == nil {
		payload = map[string]interface{}{}
	}
	setOptionalString(payload, "reason", reason)
	setOptionalString(payload, "justification", justification)
	return append(actions, api_client.AutomationV2Action{
		Type: actionType,
		Data: payload,
	})
}

func appendEmailAction(actions []api_client.AutomationV2Action, tmpl *automationV2EmailTemplateModel) []api_client.AutomationV2Action {
	if tmpl == nil {
		return actions
	}
	data := map[string]interface{}{}
	if emails := stringListToSlice(tmpl.EmailAddresses); len(emails) > 0 {
		data["email"] = emails
	}
	if tags := stringListToSlice(tmpl.AssetTagKeys); len(tags) > 0 {
		data["asset_tag_keys"] = tags
	}
	if tags := stringListToSlice(tmpl.CustomTagKeys); len(tags) > 0 {
		data["custom_tag_keys"] = tags
	}
	if !tmpl.MultiAlerts.IsNull() && !tmpl.MultiAlerts.IsUnknown() {
		data["multi_alerts"] = tmpl.MultiAlerts.ValueBool()
	}
	return append(actions, api_client.AutomationV2Action{
		Type: api_client.AutomationEmailID,
		Data: data,
	})
}

func generateV2Actions(plan *automationV2ResourceModel, apiClient *api_client.APIClient) ([]api_client.AutomationV2Action, error) {
	var actions []api_client.AutomationV2Action

	if plan.SnoozeTemplate != nil {
		actions = appendReasonJustificationAction(actions, api_client.AutomationSnoozeID,
			plan.SnoozeTemplate.Reason, plan.SnoozeTemplate.Justification,
			map[string]interface{}{"days": plan.SnoozeTemplate.Days.ValueInt64()})
	}

	if plan.AlertDismissalTemplate != nil {
		actions = appendReasonJustificationAction(actions, api_client.AutomationAlertDismissalID,
			plan.AlertDismissalTemplate.Reason, plan.AlertDismissalTemplate.Justification, nil)
	}

	if plan.AlertScoreDecreaseTemplate != nil {
		actions = appendReasonJustificationAction(actions, api_client.AutomationAlertScoreChangeID,
			plan.AlertScoreDecreaseTemplate.Reason, plan.AlertScoreDecreaseTemplate.Justification,
			map[string]interface{}{"decrease_orca_score": 1})
	}

	if plan.AlertScoreIncreaseTemplate != nil {
		actions = appendReasonJustificationAction(actions, api_client.AutomationAlertScoreChangeID,
			plan.AlertScoreIncreaseTemplate.Reason, plan.AlertScoreIncreaseTemplate.Justification,
			map[string]interface{}{"increase_orca_score": 1})
	}

	if plan.AlertScoreSpecifyTemplate != nil {
		actions = appendReasonJustificationAction(actions, api_client.AutomationAlertScoreChangeID,
			plan.AlertScoreSpecifyTemplate.Reason, plan.AlertScoreSpecifyTemplate.Justification,
			map[string]interface{}{"change_orca_score": plan.AlertScoreSpecifyTemplate.NewScore.ValueFloat64()})
	}

	externalConfigBindings := []struct {
		tmpl       *automationV2ExternalConfigTemplateModel
		actionType int32
	}{
		{plan.AwsSecurityHubTemplate, api_client.AutomationAWSSecurityHubID},
		{plan.AwsSecurityLakeTemplate, api_client.AutomationAwsSecurityLakeID},
		{plan.AwsSqsTemplate, api_client.AutomationAwsSqsID},
		{plan.AwsSnsTemplate, api_client.AutomationAwsSnsID},
		{plan.AzureSentinelTemplate, api_client.AutomationAzureSentinelID},
		{plan.ChronicleTemplate, api_client.AutomationChronicleID},
		{plan.CoralogixTemplate, api_client.AutomationCoralogixID},
		{plan.CriblTemplate, api_client.AutomationCriblID},
		{plan.GcpPubSubTemplate, api_client.AutomationGcpPubSubID},
		{plan.LinearTemplate, api_client.AutomationLinearID},
		{plan.MondayTemplate, api_client.AutomationMondayID},
		{plan.MsTeamsTemplate, api_client.AutomationMsTeamsID},
		{plan.OpsgenieTemplate, api_client.AutomationOpsgenieID},
		{plan.OpusTemplate, api_client.AutomationOpusID},
		{plan.PagerDutyTemplate, api_client.AutomationPagerDutyID},
		{plan.PantherTemplate, api_client.AutomationPantherID},
		{plan.ServiceNowIncidentsTemplate, api_client.AutomationServiceNowIncidentsID},
		{plan.ServiceNowSIIncidentsTemplate, api_client.AutomationServiceNowSIIncidentsID},
		{plan.SlackTemplate, api_client.AutomationSlackID},
		{plan.SnowflakeTemplate, api_client.AutomationSnowflakeID},
		{plan.SplunkTemplate, api_client.AutomationSplunkID},
		{plan.SumoLogicTemplate, api_client.AutomationSumoLogicID},
		{plan.TinesTemplate, api_client.AutomationTinesID},
		{plan.TorqTemplate, api_client.AutomationTorqID},
		{plan.WebhookTemplate, api_client.AutomationWebhookID},
	}
	for _, b := range externalConfigBindings {
		actions = appendExternalConfigAction(actions, b.tmpl, b.actionType)
	}

	externalConfigWithParentBindings := []struct {
		tmpl       *automationV2ExternalConfigWithParentTemplateModel
		actionType int32
	}{
		{plan.AzureDevopsTemplate, api_client.AutomationAzureDevopsID},
		{plan.JiraCloudTemplate, api_client.AutomationJiraID},
		{plan.JiraServerTemplate, api_client.AutomationJiraServerID},
	}
	for _, b := range externalConfigWithParentBindings {
		actions = appendExternalConfigWithParentAction(actions, b.tmpl, b.actionType)
	}

	if plan.ApiTokenTemplate != nil {
		token := plan.ApiTokenTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:      api_client.AutomationSiemID,
			Data:      map[string]interface{}{},
			SiemToken: &token,
		})
	}

	if plan.DatadogTemplate != nil {
		externalConfigID := plan.DatadogTemplate.ExternalConfigID.ValueString()
		actions = append(actions, api_client.AutomationV2Action{
			Type:           api_client.AutomationDatadogID,
			Data:           map[string]interface{}{"type": plan.DatadogTemplate.Type.ValueString()},
			ExternalConfig: &externalConfigID,
		})
	}

	actions = appendEmailAction(actions, plan.EmailTemplate)

	if plan.RemediationTemplate != nil {
		actions = append(actions, api_client.AutomationV2Action{
			Type: api_client.AutomationRemediationID,
			Data: map[string]interface{}{
				"remediation_action": plan.RemediationTemplate.RemediationAction.ValueString(),
			},
		})
	}

	return actions, nil
}

// stringListToSlice converts a Terraform string list into a Go slice, returning
// nil for null/unknown lists.
func stringListToSlice(l types.List) []string {
	if l.IsNull() || l.IsUnknown() {
		return nil
	}
	var out []string
	_ = l.ElementsAs(context.Background(), &out, false)
	return out
}

// --- Read-side helpers: reconstruct model from the API instance on import ---

func dataString(data map[string]interface{}, key string) types.String {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return types.StringValue(s)
		}
	}
	return types.StringNull()
}

// dataStringOrEmpty behaves like dataString but returns an empty string (not null)
// when the key is absent or the value is empty. Used only on the import-reconstruct
// path for reason/justification, whose documented "omitted" form is the empty string
// ("" is treated as omitted at the API layer, e.g. UI-exported HCL writes reason = "").
// The API never returns these once omitted, so a plain dataString would import them as
// null and perpetually diff against a `reason = ""` config. Returning "" makes imported
// state match that canonical form; a config that omits the field instead converges after
// one apply (setOptionalString drops the empty value again).
func dataStringOrEmpty(data map[string]interface{}, key string) types.String {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok {
			return types.StringValue(s)
		}
	}
	return types.StringValue("")
}

func dataInt64(data map[string]interface{}, key string) types.Int64 {
	if v, ok := data[key]; ok {
		if f, ok := v.(float64); ok {
			return types.Int64Value(int64(f))
		}
	}
	return types.Int64Null()
}

func dataFloat64(data map[string]interface{}, key string) types.Float64 {
	if v, ok := data[key]; ok {
		if f, ok := v.(float64); ok {
			return types.Float64Value(f)
		}
	}
	return types.Float64Null()
}

func dataBool(data map[string]interface{}, key string) types.Bool {
	v, ok := data[key]
	if !ok {
		return types.BoolNull()
	}
	switch t := v.(type) {
	case bool:
		return types.BoolValue(t)
	case float64: // JSON numbers decode to float64; treat non-zero as true
		return types.BoolValue(t != 0)
	case string:
		return types.BoolValue(t == "true" || t == "1")
	}
	return types.BoolNull()
}

func dataStringList(ctx context.Context, data map[string]interface{}, key string) types.List {
	raw, ok := data[key].([]interface{})
	if !ok || len(raw) == 0 {
		return types.ListNull(types.StringType)
	}
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		if s, ok := r.(string); ok {
			out = append(out, s)
		}
	}
	l, diags := types.ListValueFrom(ctx, types.StringType, out)
	if diags.HasError() {
		return types.ListNull(types.StringType)
	}
	return l
}

func extConfigTmpl(a api_client.AutomationV2Action) *automationV2ExternalConfigTemplateModel {
	id := ""
	if a.ExternalConfig != nil {
		id = *a.ExternalConfig
	}
	return &automationV2ExternalConfigTemplateModel{ExternalConfigID: types.StringValue(id)}
}

func extConfigWithParentTmpl(a api_client.AutomationV2Action) *automationV2ExternalConfigWithParentTemplateModel {
	id := ""
	if a.ExternalConfig != nil {
		id = *a.ExternalConfig
	}
	return &automationV2ExternalConfigWithParentTemplateModel{
		ExternalConfigID: types.StringValue(id),
		ParentIssueID:    dataString(a.Data, "parent_id"),
	}
}

// reconstructV2StateFromAPI rebuilds the filter, business units and every action
// template on the model from the API instance. Used on import, where there is no
// prior state to round-trip.
func reconstructV2StateFromAPI(ctx context.Context, state *automationV2ResourceModel, instance *api_client.AutomationV2) error {
	sonarJSON, err := json.Marshal(instance.Filter.SonarQuery)
	if err != nil {
		return fmt.Errorf("could not marshal sonar_query: %w", err)
	}
	state.Filter = &automationV2FilterModel{SonarQuery: types.StringValue(string(sonarJSON))}

	// apply_on_existing is a create-only (POST) flag that the API does not return.
	// Leaving it null after import makes it resolve to false on the next plan,
	// which — because it is RequiresReplace — forces a spurious destroy/create.
	// An already-existing (imported) automation is equivalent to false.
	state.ApplyOnExisting = types.BoolValue(false)

	if len(instance.BusinessUnits) > 0 {
		bu, diags := types.ListValueFrom(ctx, types.StringType, instance.BusinessUnits)
		if diags.HasError() {
			return fmt.Errorf("could not convert business_units")
		}
		state.BusinessUnits = bu
	} else {
		state.BusinessUnits = types.ListNull(types.StringType)
	}

	for _, a := range instance.Actions {
		applyV2ActionToState(ctx, state, a)
	}
	return nil
}

// extConfigTemplateSetters maps action types whose state field is a plain
// external-config template to the setter for that field. Keeping these out of
// applyV2ActionToState's switch keeps its branch count below SonarQube's limit.
var extConfigTemplateSetters = map[int32]func(*automationV2ResourceModel, *automationV2ExternalConfigTemplateModel){
	api_client.AutomationSlackID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) { s.SlackTemplate = t },
	api_client.AutomationPagerDutyID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) {
		s.PagerDutyTemplate = t
	},
	api_client.AutomationOpsgenieID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) { s.OpsgenieTemplate = t },
	api_client.AutomationSumoLogicID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) {
		s.SumoLogicTemplate = t
	},
	api_client.AutomationAzureSentinelID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) {
		s.AzureSentinelTemplate = t
	},
	api_client.AutomationSplunkID:  func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) { s.SplunkTemplate = t },
	api_client.AutomationWebhookID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) { s.WebhookTemplate = t },
	api_client.AutomationGcpPubSubID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) {
		s.GcpPubSubTemplate = t
	},
	api_client.AutomationTorqID:    func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) { s.TorqTemplate = t },
	api_client.AutomationMsTeamsID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) { s.MsTeamsTemplate = t },
	api_client.AutomationServiceNowIncidentsID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) {
		s.ServiceNowIncidentsTemplate = t
	},
	api_client.AutomationServiceNowSIIncidentsID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) {
		s.ServiceNowSIIncidentsTemplate = t
	},
	api_client.AutomationAwsSecurityLakeID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) {
		s.AwsSecurityLakeTemplate = t
	},
	api_client.AutomationSnowflakeID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) {
		s.SnowflakeTemplate = t
	},
	api_client.AutomationChronicleID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) {
		s.ChronicleTemplate = t
	},
	api_client.AutomationCriblID:  func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) { s.CriblTemplate = t },
	api_client.AutomationTinesID:  func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) { s.TinesTemplate = t },
	api_client.AutomationAwsSqsID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) { s.AwsSqsTemplate = t },
	api_client.AutomationAwsSnsID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) { s.AwsSnsTemplate = t },
	api_client.AutomationOpusID:   func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) { s.OpusTemplate = t },
	api_client.AutomationCoralogixID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) {
		s.CoralogixTemplate = t
	},
	api_client.AutomationAWSSecurityHubID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) {
		s.AwsSecurityHubTemplate = t
	},
	api_client.AutomationMondayID:  func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) { s.MondayTemplate = t },
	api_client.AutomationLinearID:  func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) { s.LinearTemplate = t },
	api_client.AutomationPantherID: func(s *automationV2ResourceModel, t *automationV2ExternalConfigTemplateModel) { s.PantherTemplate = t },
}

// field of the model. Split out of reconstructV2StateFromAPI to keep that
// function's cognitive complexity low.
func applyV2ActionToState(ctx context.Context, state *automationV2ResourceModel, a api_client.AutomationV2Action) {
	if set, ok := extConfigTemplateSetters[a.Type]; ok {
		set(state, extConfigTmpl(a))
		return
	}
	switch a.Type {
	case api_client.AutomationAlertDismissalID:
		state.AlertDismissalTemplate = &automationV2AlertDismissalTemplateModel{
			Reason:        dataStringOrEmpty(a.Data, "reason"),
			Justification: dataStringOrEmpty(a.Data, "justification"),
		}
	case api_client.AutomationAlertScoreChangeID:
		applyV2AlertScoreChangeToState(state, a)
	case api_client.AutomationSnoozeID:
		state.SnoozeTemplate = &automationV2SnoozeTemplateModel{
			Days:          dataInt64(a.Data, "days"),
			Reason:        dataStringOrEmpty(a.Data, "reason"),
			Justification: dataStringOrEmpty(a.Data, "justification"),
		}
	case api_client.AutomationEmailID:
		state.EmailTemplate = &automationV2EmailTemplateModel{
			EmailAddresses: dataStringList(ctx, a.Data, "email"),
			MultiAlerts:    dataBool(a.Data, "multi_alerts"),
			AssetTagKeys:   dataStringList(ctx, a.Data, "asset_tag_keys"),
			CustomTagKeys:  dataStringList(ctx, a.Data, "custom_tag_keys"),
		}
	case api_client.AutomationRemediationID:
		state.RemediationTemplate = &automationV2RemediationTemplateModel{
			RemediationAction: dataString(a.Data, "remediation_action"),
		}
	case api_client.AutomationSiemID:
		token := ""
		if a.SiemToken != nil {
			token = *a.SiemToken
		}
		state.ApiTokenTemplate = &automationV2ExternalConfigTemplateModel{
			ExternalConfigID: types.StringValue(token),
		}
	case api_client.AutomationDatadogID:
		id := ""
		if a.ExternalConfig != nil {
			id = *a.ExternalConfig
		}
		state.DatadogTemplate = &automationV2DatadogTemplateModel{
			ExternalConfigID: types.StringValue(id),
			Type:             dataString(a.Data, "type"),
		}
	case api_client.AutomationAzureDevopsID:
		state.AzureDevopsTemplate = extConfigWithParentTmpl(a)
	case api_client.AutomationJiraID:
		state.JiraCloudTemplate = extConfigWithParentTmpl(a)
	case api_client.AutomationJiraServerID:
		state.JiraServerTemplate = extConfigWithParentTmpl(a)
	}
}

// applyV2AlertScoreChangeToState maps an alert-score-change action onto the
// matching decrease/increase/specify template field of the model.
func applyV2AlertScoreChangeToState(state *automationV2ResourceModel, a api_client.AutomationV2Action) {
	switch {
	case func() bool { _, ok := a.Data["decrease_orca_score"]; return ok }():
		state.AlertScoreDecreaseTemplate = &automationV2AlertScoreDecreaseTemplateModel{
			Reason:        dataStringOrEmpty(a.Data, "reason"),
			Justification: dataStringOrEmpty(a.Data, "justification"),
		}
	case func() bool { _, ok := a.Data["increase_orca_score"]; return ok }():
		state.AlertScoreIncreaseTemplate = &automationV2AlertScoreIncreaseTemplateModel{
			Reason:        dataStringOrEmpty(a.Data, "reason"),
			Justification: dataStringOrEmpty(a.Data, "justification"),
		}
	default:
		state.AlertScoreSpecifyTemplate = &automationV2AlertScoreSpecifyTemplateModel{
			NewScore:      dataFloat64(a.Data, "change_orca_score"),
			Reason:        dataStringOrEmpty(a.Data, "reason"),
			Justification: dataStringOrEmpty(a.Data, "justification"),
		}
	}
}

// applyPriority moves the automation to the requested evaluation-order
// position and returns the priority the server actually assigned. The server
// silently clamps values above the automation count, so callers must compare
// the returned value with the requested one and surface a diagnostic on
// mismatch.
func (r *automationV2Resource) applyPriority(id string, requested int64) (int64, error) {
	instance, err := r.apiClient.SetAutomationV2Priority(id, requested)
	if err != nil {
		return 0, err
	}
	if instance == nil || instance.Priority == nil {
		return 0, fmt.Errorf("priority endpoint returned no priority value")
	}
	return *instance.Priority, nil
}

// clampErrorDetail formats the diagnostic message shown to users when the
// server clamps a requested priority to a lower value than requested.
func clampErrorDetail(requested, actual int64) string {
	return fmt.Sprintf(
		"priority %d exceeds the number of automations; the server placed the automation at priority %d. "+
			"The automation is tracked in state — lower priority in the configuration and re-apply.",
		requested, actual)
}

// refreshPriority updates the model's priority from the API instance, but only
// when priority is already tracked (non-null) in state. Untracked priority
// stays null so configurations that never set it see no drift noise from
// external reordering.
func refreshPriority(state *automationV2ResourceModel, instance *api_client.AutomationV2) {
	if state.Priority.IsNull() {
		return
	}
	if instance == nil {
		state.Priority = types.Int64Null()
		return
	}
	state.Priority = types.Int64PointerValue(instance.Priority)
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

	applyOnExisting := plan.ApplyOnExisting.ValueBool()
	instance, err := r.apiClient.CreateAutomationV2(createReq, applyOnExisting)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Automation V2",
			"Could not create Automation V2, unexpected error: "+err.Error(),
		)
		return
	}

	if instance == nil {
		resp.Diagnostics.AddError(
			"Error creating Automation V2",
			"Could not create Automation V2: received nil instance from API",
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.OrganizationID = types.StringValue(instance.OrganizationID)
	plan.ApplyOnExisting = types.BoolValue(applyOnExisting)

	normalizedStatus := instance.Status
	if normalizedStatus == "success" {
		normalizedStatus = "enabled"
	}
	plan.Status = types.StringValue(normalizedStatus)

	if instance.EndTime != "" {
		plan.EndTime = types.StringValue(instance.EndTime)
	}

	if r.applyPlanPriorityOnCreate(ctx, &plan, instance.ID, resp) {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// applyPlanPriorityOnCreate sets the requested priority on a just-created
// automation, if any was requested. It writes state before returning true on
// error so the automation is tracked rather than orphaned. Split out of
// Create to keep that function's cognitive complexity low.
func (r *automationV2Resource) applyPlanPriorityOnCreate(ctx context.Context, plan *automationV2ResourceModel, instanceID string, resp *resource.CreateResponse) bool {
	if plan.Priority.IsNull() || plan.Priority.IsUnknown() {
		return false
	}
	requested := plan.Priority.ValueInt64()
	actual, err := r.applyPriority(instanceID, requested)
	if err != nil {
		plan.Priority = types.Int64Null()
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError(
			"Error setting Automation V2 priority",
			fmt.Sprintf("Automation %s was created, but setting priority failed: %s", instanceID, err.Error()),
		)
		return true
	}
	if actual != requested {
		plan.Priority = types.Int64Value(actual)
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("Automation V2 priority out of range", clampErrorDetail(requested, actual))
		return true
	}
	return false
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

	if instance == nil {
		resp.Diagnostics.AddError(
			"Error reading Automation V2",
			fmt.Sprintf("Could not read Automation V2 ID %s: received nil instance from API", state.ID.ValueString()),
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

	// On import there is no prior state for the filter or action templates
	// (filter is Required, so a nil filter means this Read follows an import).
	// Rebuild them from the API so `terraform plan` is clean. On a normal
	// refresh the prior state already round-trips the user's exact JSON/template
	// shape, so it is left untouched to avoid spurious diffs.
	if state.Filter == nil {
		if err := reconstructV2StateFromAPI(ctx, &state, instance); err != nil {
			resp.Diagnostics.AddError(
				"Error reading Automation V2",
				fmt.Sprintf("Could not reconstruct state for ID %s: %s", state.ID.ValueString(), err.Error()),
			)
			return
		}
	}

	refreshPriority(&state, instance)

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

	updatedInstance, err := r.apiClient.UpdateAutomationV2(plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Automation V2",
			fmt.Sprintf("Could not update Automation V2, unexpected error: %s", err.Error()),
		)
		return
	}

	if updatedInstance == nil {
		resp.Diagnostics.AddError(
			"Error updating Automation V2",
			fmt.Sprintf("Could not update Automation V2 ID %s: received nil instance from API", plan.ID.ValueString()),
		)
		return
	}

	verifyInstance, err := r.apiClient.GetAutomationV2(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Automation V2",
			"Could not read Automation V2 ID: "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	if verifyInstance == nil {
		resp.Diagnostics.AddError(
			"Error reading Automation V2",
			fmt.Sprintf("Could not read Automation V2 ID %s: received nil instance from API during verification", plan.ID.ValueString()),
		)
		return
	}

	// Refresh computed fields from the API response to prevent perpetual diffs
	plan.ID = types.StringValue(updatedInstance.ID)
	plan.OrganizationID = types.StringValue(updatedInstance.OrganizationID)

	// Normalize status field (API returns "success" for enabled automations)
	normalizedStatus := updatedInstance.Status
	if normalizedStatus == "success" {
		normalizedStatus = "enabled"
	}
	plan.Status = types.StringValue(normalizedStatus)

	// Refresh end_time from API response (server might normalize the format)
	if updatedInstance.EndTime != "" {
		plan.EndTime = types.StringValue(updatedInstance.EndTime)
	} else {
		plan.EndTime = types.StringNull()
	}

	if r.applyPlanPriorityOnUpdate(ctx, &plan, verifyInstance, resp) {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// applyPlanPriorityOnUpdate PUTs the requested priority only when it differs
// from the automation's current server-side priority, and does nothing when
// the plan's priority is null (user stopped tracking it: no API call, no
// error). Split out of Update to keep that function's cognitive complexity
// low.
func (r *automationV2Resource) applyPlanPriorityOnUpdate(ctx context.Context, plan *automationV2ResourceModel, verifyInstance *api_client.AutomationV2, resp *resource.UpdateResponse) bool {
	if plan.Priority.IsNull() || plan.Priority.IsUnknown() {
		return false
	}
	requested := plan.Priority.ValueInt64()
	if verifyInstance.Priority != nil && *verifyInstance.Priority == requested {
		return false
	}
	actual, err := r.applyPriority(plan.ID.ValueString(), requested)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error setting Automation V2 priority",
			fmt.Sprintf("Could not set priority for Automation V2 ID %s: %s", plan.ID.ValueString(), err.Error()),
		)
		return true
	}
	if actual != requested {
		plan.Priority = types.Int64Value(actual)
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("Automation V2 priority out of range", clampErrorDetail(requested, actual))
		return true
	}
	return false
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
