package jira_cloud_template

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &jiraCloudResource{}
	_ resource.ResourceWithConfigure   = &jiraCloudResource{}
	_ resource.ResourceWithImportState = &jiraCloudResource{}
)

type jiraCloudResource struct {
	apiClient *api_client.APIClient
}

type jiraCloudResourceModel struct {
	ID                             types.String `tfsdk:"id"`
	TemplateName                   types.String `tfsdk:"template_name"`
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
	IsEnabled                      types.Bool   `tfsdk:"is_enabled"`
	IsDefault                      types.Bool   `tfsdk:"is_default"`
	BusinessUnits                  types.Set    `tfsdk:"business_units"`
}

func NewJiraCloudTemplateResource() resource.Resource {
	return &jiraCloudResource{}
}

func (r *jiraCloudResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_jira_cloud_template"
}

func (r *jiraCloudResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *jiraCloudResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage a Jira Cloud template in Orca. Creates an external service config of `service_name = \"jira\"` linked to an existing Jira Cloud OAuth resource. Holds the project, issue-type, field-mapping, and status-mapping settings used when Orca opens Jira issues.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Orca external service config identifier (UUID).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"template_name": schema.StringAttribute{
				Required:    true,
				Description: "Template name as shown in the Orca UI and used as the URL key for update/delete. Changing this forces a new resource.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_id": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the Jira Cloud OAuth resource that carries the credentials (look it up in the Orca UI under Integrations → Jira Cloud).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"resource_url": schema.StringAttribute{
				Required:    true,
				Description: "Jira tenant URL (for example, `https://acme.atlassian.net`).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "Jira project ID Orca opens issues in.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"issue_type_id": schema.StringAttribute{
				Required:    true,
				Description: "Jira issue type ID for the main ticket.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
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
			"is_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"is_default": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"business_units": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Optional set of Orca business unit IDs that may use this template. Orca only accepts this value at create time — changes force Terraform to replace the template.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *jiraCloudResource) buildPayload(ctx context.Context, plan jiraCloudResourceModel, diags *diag.Diagnostics) api_client.JiraCloudTemplate {
	cfg := api_client.JiraCloudTemplateConfig{
		ResourceID:         plan.ResourceID.ValueString(),
		ResourceURL:        plan.ResourceURL.ValueString(),
		ProjectID:          plan.ProjectID.ValueString(),
		IssueTypeID:        plan.IssueTypeID.ValueString(),
		SubtaskIssueTypeID: plan.SubtaskIssueTypeID.ValueString(),
	}

	mapping, mappingDiags := common.DecodeJSONField(plan.MappingJSON, "mapping_json")
	diags.Append(mappingDiags...)
	cfg.Mapping = mapping

	alertStatus, alertDiags := common.DecodeJSONField(plan.AlertStatusMappingJSON, "alert_status_mapping_json")
	diags.Append(alertDiags...)
	cfg.AlertStatusMapping = alertStatus

	ticketStatus, ticketDiags := common.DecodeJSONField(plan.TicketStatusMappingJSON, "ticket_status_mapping_json")
	diags.Append(ticketDiags...)
	cfg.TicketStatusMapping = ticketStatus

	subAlertStatus, subAlertDiags := common.DecodeJSONField(plan.SubtaskAlertStatusMappingJSON, "subtask_alert_status_mapping_json")
	diags.Append(subAlertDiags...)
	cfg.SubtaskAlertStatusMapping = subAlertStatus

	subTicketStatus, subTicketDiags := common.DecodeJSONField(plan.SubtaskTicketStatusMappingJSON, "subtask_ticket_status_mapping_json")
	diags.Append(subTicketDiags...)
	cfg.SubtaskTicketStatusMapping = subTicketStatus

	payload := api_client.JiraCloudTemplate{
		TemplateName: plan.TemplateName.ValueString(),
		IsEnabled:    plan.IsEnabled.ValueBool(),
		IsDefault:    plan.IsDefault.ValueBool(),
		Config:       cfg,
	}

	if !plan.BusinessUnits.IsNull() && !plan.BusinessUnits.IsUnknown() {
		var bus []string
		diags.Append(plan.BusinessUnits.ElementsAs(ctx, &bus, false)...)
		payload.BusinessUnits = bus
	}

	return payload
}

func (r *jiraCloudResource) applyAPIResponse(ctx context.Context, plan *jiraCloudResourceModel, apiObj *api_client.JiraCloudTemplate, diags *diag.Diagnostics) {
	plan.ID = types.StringValue(apiObj.ID)
	plan.IsEnabled = types.BoolValue(apiObj.IsEnabled)
	plan.IsDefault = types.BoolValue(apiObj.IsDefault)
	if apiObj.TemplateName != "" {
		plan.TemplateName = types.StringValue(apiObj.TemplateName)
	}

	if apiObj.Config.ResourceID != "" {
		plan.ResourceID = types.StringValue(apiObj.Config.ResourceID)
	}
	if apiObj.Config.ResourceURL != "" {
		plan.ResourceURL = types.StringValue(apiObj.Config.ResourceURL)
	}
	if apiObj.Config.ProjectID != "" {
		plan.ProjectID = types.StringValue(apiObj.Config.ProjectID)
	}
	if apiObj.Config.IssueTypeID != "" {
		plan.IssueTypeID = types.StringValue(apiObj.Config.IssueTypeID)
	}
	if apiObj.Config.SubtaskIssueTypeID != "" {
		plan.SubtaskIssueTypeID = types.StringValue(apiObj.Config.SubtaskIssueTypeID)
	}

	mapping, mappingDiags := common.EncodeJSONField(apiObj.Config.Mapping, plan.MappingJSON)
	diags.Append(mappingDiags...)
	plan.MappingJSON = mapping

	alertStatus, alertDiags := common.EncodeJSONField(apiObj.Config.AlertStatusMapping, plan.AlertStatusMappingJSON)
	diags.Append(alertDiags...)
	plan.AlertStatusMappingJSON = alertStatus

	ticketStatus, ticketDiags := common.EncodeJSONField(apiObj.Config.TicketStatusMapping, plan.TicketStatusMappingJSON)
	diags.Append(ticketDiags...)
	plan.TicketStatusMappingJSON = ticketStatus

	subAlertStatus, subAlertDiags := common.EncodeJSONField(apiObj.Config.SubtaskAlertStatusMapping, plan.SubtaskAlertStatusMappingJSON)
	diags.Append(subAlertDiags...)
	plan.SubtaskAlertStatusMappingJSON = subAlertStatus

	subTicketStatus, subTicketDiags := common.EncodeJSONField(apiObj.Config.SubtaskTicketStatusMapping, plan.SubtaskTicketStatusMappingJSON)
	diags.Append(subTicketDiags...)
	plan.SubtaskTicketStatusMappingJSON = subTicketStatus

	bus, busDiags := common.BusinessUnitsFromAPI(ctx, apiObj.BusinessUnits, plan.BusinessUnits)
	diags.Append(busDiags...)
	plan.BusinessUnits = bus
}

func (r *jiraCloudResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating Jira Cloud template", "API client not configured.")
		return
	}

	var plan jiraCloudResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := r.buildPayload(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.apiClient.CreateJiraCloudTemplate(payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Jira Cloud template",
			fmt.Sprintf("Could not create Jira Cloud template: %s", err.Error()),
		)
		return
	}

	r.applyAPIResponse(ctx, &plan, created, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *jiraCloudResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state jiraCloudResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.apiClient.GetJiraCloudTemplate(state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Jira Cloud template",
			fmt.Sprintf("Could not read Jira Cloud template %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	if current == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.applyAPIResponse(ctx, &state, current, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *jiraCloudResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan jiraCloudResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state jiraCloudResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := r.buildPayload(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.apiClient.UpdateJiraCloudTemplate(state.TemplateName.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Jira Cloud template",
			fmt.Sprintf("Could not update Jira Cloud template %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	r.applyAPIResponse(ctx, &plan, updated, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *jiraCloudResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state jiraCloudResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteJiraCloudTemplate(state.TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Jira Cloud template",
			fmt.Sprintf("Could not delete Jira Cloud template %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *jiraCloudResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
