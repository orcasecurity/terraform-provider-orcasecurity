package jira_cloud_template

import (
	"context"
	"encoding/json"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// state is the per-variant Terraform model. CommonFieldsWithBU carries id / template_name /
// is_enabled / is_default / business_units plus the GetCommon/SetCommon glue the generic spec
// needs.
type state struct {
	cc.CommonFieldsWithBU
	ResourceID                     types.String `tfsdk:"resource_id"`
	ResourceURL                    types.String `tfsdk:"resource_url"`
	ProjectID                      types.String `tfsdk:"project_id"`
	IssueTypeID                    types.String `tfsdk:"issue_type_id"`
	SubtaskIssueTypeID             types.String `tfsdk:"subtask_issue_type_id"`
	MappingJSON                    types.String `tfsdk:"mapping_json"`
	AlertStatusMappingJSON         types.String `tfsdk:"alert_status_mapping_json"`
	TicketStatusMappingJSON        types.String `tfsdk:"ticket_status_mapping_json"`
	SubtaskAlertStatusMappingJSON  types.String `tfsdk:"subtask_alert_status_mapping_json"`
	SubtaskTicketStatusMappingJSON types.String `tfsdk:"subtask_ticket_status_mapping_json"`
}

func variantAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"resource_id": schema.StringAttribute{
			Required:    true,
			Description: "UUID of the Jira Cloud OAuth resource that carries the credentials (look it up in the Orca UI under Integrations → Jira Cloud).",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"resource_url": schema.StringAttribute{
			Required:    true,
			Description: "Jira tenant URL (for example, `https://acme.atlassian.net`).",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"project_id": schema.StringAttribute{
			Required:    true,
			Description: "Jira project ID Orca opens issues in.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"issue_type_id": schema.StringAttribute{
			Required:    true,
			Description: "Jira issue type ID for the main ticket.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"subtask_issue_type_id": schema.StringAttribute{
			Optional:    true,
			Description: "Jira issue type ID for sub-task tickets.",
		},
		"mapping_json": schema.StringAttribute{
			Optional:    true,
			Description: "JSON-encoded `mapping` object. Each key is a Jira field name; each value is a list of `{ \"orca\": \"<alert_field>\" }` or `{ \"value\": \"<literal>\" }` entries. Multiple entries are concatenated when the Jira field accepts a single value.",
		},
		"alert_status_mapping_json": schema.StringAttribute{
			Optional:    true,
			Description: "JSON-encoded `alert_status_mapping` — maps Orca alert statuses to Jira workflow status IDs (for example, `{\"in_progress\": \"10001\"}`).",
		},
		"ticket_status_mapping_json": schema.StringAttribute{
			Optional:    true,
			Description: "JSON-encoded `ticket_status_mapping` — maps Jira workflow status IDs back to Orca alert state changes (for example, `{\"10000\": {\"status\": \"snoozed\", \"snooze_days\": 1}}`).",
		},
		"subtask_alert_status_mapping_json": schema.StringAttribute{
			Optional:    true,
			Description: "JSON-encoded `subtask_alert_status_mapping` — same shape as `alert_status_mapping_json`, applied to sub-task tickets.",
		},
		"subtask_ticket_status_mapping_json": schema.StringAttribute{
			Optional:    true,
			Description: "JSON-encoded `subtask_ticket_status_mapping` — same shape as `ticket_status_mapping_json`, applied to sub-task tickets.",
		},
		// Override the base business_units attribute: Orca only accepts this value at create time
		// (updates are rejected with "You can't change business units"), so a change forces replace.
		"business_units": schema.SetAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "Optional set of Orca business unit IDs that may use this template. Orca only accepts this value at create time — changes force Terraform to replace the template.",
			PlanModifiers: []planmodifier.Set{
				setplanmodifier.RequiresReplace(),
			},
		},
	}
}

// decodeMappings pulls the five JSON-string fields off the plan into the API config.
func decodeMappings(s *state, cfg *api_client.JiraCloudTemplateConfig, diags *diag.Diagnostics) {
	for _, m := range []struct {
		src   types.String
		field string
		dst   *json.RawMessage
	}{
		{s.MappingJSON, "mapping_json", &cfg.Mapping},
		{s.AlertStatusMappingJSON, "alert_status_mapping_json", &cfg.AlertStatusMapping},
		{s.TicketStatusMappingJSON, "ticket_status_mapping_json", &cfg.TicketStatusMapping},
		{s.SubtaskAlertStatusMappingJSON, "subtask_alert_status_mapping_json", &cfg.SubtaskAlertStatusMapping},
		{s.SubtaskTicketStatusMappingJSON, "subtask_ticket_status_mapping_json", &cfg.SubtaskTicketStatusMapping},
	} {
		raw, d := common.DecodeJSONField(m.src, m.field)
		diags.Append(d...)
		*m.dst = raw
	}
}

// encodeMappings writes the five JSON config fields from the API response back onto state,
// preserving each field's planned whitespace shape via EncodeJSONField.
func encodeMappings(s *state, cfg *api_client.JiraCloudTemplateConfig, diags *diag.Diagnostics) {
	for _, m := range []struct {
		raw json.RawMessage
		dst *types.String
	}{
		{cfg.Mapping, &s.MappingJSON},
		{cfg.AlertStatusMapping, &s.AlertStatusMappingJSON},
		{cfg.TicketStatusMapping, &s.TicketStatusMappingJSON},
		{cfg.SubtaskAlertStatusMapping, &s.SubtaskAlertStatusMappingJSON},
		{cfg.SubtaskTicketStatusMapping, &s.SubtaskTicketStatusMappingJSON},
	} {
		encoded, d := common.EncodeJSONField(m.raw, *m.dst)
		diags.Append(d...)
		*m.dst = encoded
	}
}

func NewJiraCloudTemplateResource() resource.Resource {
	return cc.New(cc.Spec[api_client.JiraCloudTemplate]{
		TypeNameSuffix:        "_integration_jira_cloud_template",
		UIName:                "Jira Cloud template",
		Description:           "Manage a Jira Cloud template in Orca. Creates an external service config of `service_name = \"jira\"` linked to an existing Jira Cloud OAuth resource. Holds the project, issue-type, field-mapping, and status-mapping settings used when Orca opens Jira issues.",
		SupportsBusinessUnits: true,
		VariantAttributes:     variantAttributes(),
		NewState:              func() cc.State { return &state{} },
		BuildPayload: func(ctx context.Context, st cc.State, diags *diag.Diagnostics) api_client.JiraCloudTemplate {
			s := st.(*state)
			cfg := api_client.JiraCloudTemplateConfig{
				ResourceID:         s.ResourceID.ValueString(),
				ResourceURL:        s.ResourceURL.ValueString(),
				ProjectID:          s.ProjectID.ValueString(),
				IssueTypeID:        s.IssueTypeID.ValueString(),
				SubtaskIssueTypeID: s.SubtaskIssueTypeID.ValueString(),
			}
			decodeMappings(s, &cfg, diags)
			return api_client.JiraCloudTemplate{
				TemplateName:  s.TemplateName.ValueString(),
				IsEnabled:     s.IsEnabled.ValueBool(),
				IsDefault:     s.IsDefault.ValueBool(),
				Config:        cfg,
				BusinessUnits: common.BusinessUnitsToAPI(ctx, s.BusinessUnits, diags),
			}
		},
		Extract: func(o *api_client.JiraCloudTemplate, st cc.State, diags *diag.Diagnostics) cc.APIObject {
			s := st.(*state)
			if o.Config.ResourceID != "" {
				s.ResourceID = types.StringValue(o.Config.ResourceID)
			}
			if o.Config.ResourceURL != "" {
				s.ResourceURL = types.StringValue(o.Config.ResourceURL)
			}
			if o.Config.ProjectID != "" {
				s.ProjectID = types.StringValue(o.Config.ProjectID)
			}
			if o.Config.IssueTypeID != "" {
				s.IssueTypeID = types.StringValue(o.Config.IssueTypeID)
			}
			if o.Config.SubtaskIssueTypeID != "" {
				s.SubtaskIssueTypeID = types.StringValue(o.Config.SubtaskIssueTypeID)
			}
			encodeMappings(s, &o.Config, diags)
			return cc.APIObject{
				ID:            o.ID,
				TemplateName:  o.TemplateName,
				IsEnabled:     o.IsEnabled,
				IsDefault:     o.IsDefault,
				BusinessUnits: o.BusinessUnits,
			}
		},
		Create: (*api_client.APIClient).CreateJiraCloudTemplate,
		Get:    (*api_client.APIClient).GetJiraCloudTemplate,
		Update: (*api_client.APIClient).UpdateJiraCloudTemplate,
		Delete: (*api_client.APIClient).DeleteJiraCloudTemplate,
	})
}
