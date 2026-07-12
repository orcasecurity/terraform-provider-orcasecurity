package jira_cloud_template

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
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
	ResourceID                     types.String         `tfsdk:"resource_id"`
	ResourceURL                    types.String         `tfsdk:"resource_url"`
	ProjectID                      types.String         `tfsdk:"project_id"`
	IssueTypeID                    types.String         `tfsdk:"issue_type_id"`
	SubtaskIssueTypeID             types.String         `tfsdk:"subtask_issue_type_id"`
	MappingJSON                    common.OrcaMapping   `tfsdk:"mapping_json"`
	AlertStatusMappingJSON         jsontypes.Normalized `tfsdk:"alert_status_mapping_json"`
	TicketStatusMappingJSON        jsontypes.Normalized `tfsdk:"ticket_status_mapping_json"`
	SubtaskAlertStatusMappingJSON  jsontypes.Normalized `tfsdk:"subtask_alert_status_mapping_json"`
	SubtaskTicketStatusMappingJSON jsontypes.Normalized `tfsdk:"subtask_ticket_status_mapping_json"`
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
			Required:    true,
			CustomType:  common.OrcaMappingType{},
			Description: "JSON-encoded `mapping` object. Each key is a Jira field name; each value is a list of `{ \"orca\": \"<alert_field>\" }` or `{ \"value\": \"<literal>\" }` entries. Multiple entries are concatenated when the Jira field accepts a single value.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"alert_status_mapping_json": schema.StringAttribute{
			Optional:    true,
			CustomType:  jsontypes.NormalizedType{},
			Description: "JSON-encoded `alert_status_mapping` — maps Orca alert statuses to Jira workflow status IDs (for example, `{\"in_progress\": \"10001\"}`).",
		},
		"ticket_status_mapping_json": schema.StringAttribute{
			Optional:    true,
			CustomType:  jsontypes.NormalizedType{},
			Description: "JSON-encoded `ticket_status_mapping` — maps Jira workflow status IDs back to Orca alert state changes (for example, `{\"10000\": {\"status\": \"snoozed\", \"snooze_days\": 1}}`).",
		},
		"subtask_alert_status_mapping_json": schema.StringAttribute{
			Optional:    true,
			CustomType:  jsontypes.NormalizedType{},
			Description: "JSON-encoded `subtask_alert_status_mapping` — same shape as `alert_status_mapping_json`, applied to sub-task tickets.",
		},
		"subtask_ticket_status_mapping_json": schema.StringAttribute{
			Optional:    true,
			CustomType:  jsontypes.NormalizedType{},
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

// decodeMappings pulls the five JSON-string fields off the plan into the API config. The
// field `mapping` uses the bare-string orca shorthand; the status maps are plain JSON.
func decodeMappings(s *state, cfg *api_client.JiraCloudTemplateConfig, diags *diag.Diagnostics) {
	mapping, d := common.DecodeOrcaMappingField(s.MappingJSON, "mapping_json")
	diags.Append(d...)
	cfg.Mapping = mapping
	common.DecodeJSONFields([]common.JSONFieldDecode{
		{Src: s.AlertStatusMappingJSON, Field: "alert_status_mapping_json", Dst: &cfg.AlertStatusMapping},
		{Src: s.TicketStatusMappingJSON, Field: "ticket_status_mapping_json", Dst: &cfg.TicketStatusMapping},
		{Src: s.SubtaskAlertStatusMappingJSON, Field: "subtask_alert_status_mapping_json", Dst: &cfg.SubtaskAlertStatusMapping},
		{Src: s.SubtaskTicketStatusMappingJSON, Field: "subtask_ticket_status_mapping_json", Dst: &cfg.SubtaskTicketStatusMapping},
	}, diags)
}

// encodeMappings writes the five JSON config fields from the API response back onto state
// verbatim. The mapping types' semantic equality absorbs whitespace/key-order and the orca
// shorthand, so the framework keeps the user's HCL form and no diff appears.
func encodeMappings(s *state, cfg *api_client.JiraCloudTemplateConfig, diags *diag.Diagnostics) {
	mapping, d := common.EncodeOrcaMappingField(cfg.Mapping, s.MappingJSON)
	diags.Append(d...)
	s.MappingJSON = mapping
	common.EncodeJSONFields([]common.JSONFieldEncode{
		{Raw: cfg.AlertStatusMapping, Dst: &s.AlertStatusMappingJSON},
		{Raw: cfg.TicketStatusMapping, Dst: &s.TicketStatusMappingJSON},
		{Raw: cfg.SubtaskAlertStatusMapping, Dst: &s.SubtaskAlertStatusMappingJSON},
		{Raw: cfg.SubtaskTicketStatusMapping, Dst: &s.SubtaskTicketStatusMappingJSON},
	}, diags)
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
