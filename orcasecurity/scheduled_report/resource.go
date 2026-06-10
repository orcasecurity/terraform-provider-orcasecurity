package scheduled_report

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &scheduledReportResource{}
	_ resource.ResourceWithConfigure   = &scheduledReportResource{}
	_ resource.ResourceWithImportState = &scheduledReportResource{}
)

var exportTimeRegex = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d:[0-5]\d$`)

var reportTypes = []string{
	"executive", "inventory", "compliance", "alerts", "vulnerabilities",
	"discovery", "subscription", "cloud_accounts", "audit_logs", "custom_report",
	"compliance_framework", "compliance_asset", "compliance_summary", "users",
	"alerts_svl", "fedramp_inventory", "fedramp_appendix_m", "discovery_svl",
	"discovery_obs", "compliance_asset_svl", "dashboard", "cdr_events", "system_logs",
}

var reportFormats = []string{
	"csv", "json", "pdf", "xlsx", "html", "cyclone_dx_json", "spdx_json",
}

var reportRecurrences = []string{
	"daily", "weekly", "monthly", "quarterly", "once",
}

var reportStatusToInt = map[string]int{
	"created":  api_client.ScheduledReportStatusCreated,
	"active":   api_client.ScheduledReportStatusActive,
	"disabled": api_client.ScheduledReportStatusDisabled,
	"archived": api_client.ScheduledReportStatusArchived,
}

func reportStatusToString(status *int) string {
	if status == nil {
		return "active"
	}
	for name, value := range reportStatusToInt {
		if value == *status {
			return name
		}
	}
	return fmt.Sprintf("%d", *status)
}

type scheduledReportResource struct {
	apiClient *api_client.APIClient
}

type slackChannelResourceModel struct {
	Workspace types.String `tfsdk:"workspace"`
	Channel   types.String `tfsdk:"channel"`
}

type scheduledReportResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Type            types.String `tfsdk:"type"`
	Format          types.String `tfsdk:"format"`
	Recurrence      types.String `tfsdk:"recurrence"`
	FirstReportDate types.String `tfsdk:"first_report_date"`
	ExportTime      types.String `tfsdk:"export_time"`
	Status          types.String `tfsdk:"status"`

	Columns          types.List   `tfsdk:"columns"`
	DSLFilter        types.String `tfsdk:"dsl_filter"`
	SonarQuery       types.String `tfsdk:"sonar_query"`
	SonarQueryParams types.String `tfsdk:"sonar_query_params"`
	QueryFilters     types.String `tfsdk:"query_filters"`
	Config           types.String `tfsdk:"config"`
	S3Path           types.String `tfsdk:"s3_path"`

	RecipientsEmails   types.List   `tfsdk:"recipients_emails"`
	CustomEmailSubject types.String `tfsdk:"custom_email_subject"`
	CustomEmailContent types.String `tfsdk:"custom_email_content"`

	ShareToSlack types.Bool                 `tfsdk:"share_to_slack"`
	SlackChannel *slackChannelResourceModel `tfsdk:"slack_channel"`

	ShareToBucket types.Bool   `tfsdk:"share_to_bucket"`
	Bucket        types.String `tfsdk:"bucket"`

	ShareToAzureBlob   types.Bool   `tfsdk:"share_to_azure_blob"`
	AzureBlobContainer types.String `tfsdk:"azure_blob_container"`

	ShareToGoogleCloudStorage  types.Bool   `tfsdk:"share_to_google_cloud_storage"`
	GoogleCloudStorageTemplate types.String `tfsdk:"google_cloud_storage_template"`

	ShareToSnowflake  types.Bool   `tfsdk:"share_to_snowflake"`
	SnowflakeTemplate types.String `tfsdk:"snowflake_template"`
}

func NewScheduledReportResource() resource.Resource {
	return &scheduledReportResource{}
}

func (r *scheduledReportResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduled_report"
}

func (r *scheduledReportResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *scheduledReportResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *scheduledReportResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a scheduled report. Scheduled reports are generated periodically and delivered " +
			"via email, Slack, S3, Azure Blob Storage, Google Cloud Storage or Snowflake.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Scheduled report ID.",
			},
			"name": schema.StringAttribute{
				Description: "Scheduled report name. Must be unique within your organization.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(1024),
				},
			},
			"type": schema.StringAttribute{
				Description: fmt.Sprintf("Report type. Valid values are: `%v`. ", reportTypes) +
					"Prefer the `alerts_svl` and `discovery_svl` types together with `sonar_query`. " +
					"The `executive`, `inventory`, `compliance`, `alerts` and `vulnerabilities` types are deprecated and may be rejected by the API.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(reportTypes...),
				},
			},
			"format": schema.StringAttribute{
				Description: fmt.Sprintf("Report file format. Valid values are: `%v`.", reportFormats),
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(reportFormats...),
				},
			},
			"recurrence": schema.StringAttribute{
				Description: fmt.Sprintf("Report generation frequency. Valid values are: `%v`.", reportRecurrences),
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(reportRecurrences...),
				},
			},
			"first_report_date": schema.StringAttribute{
				Description: "Date and time (UTC) when the first report should be generated. ISO 8601 format, for example `2026-01-01T00:00:00Z`.",
				Required:    true,
			},
			"export_time": schema.StringAttribute{
				Description: "Time of day (UTC) when recurring reports are generated, in `HH:MM:SS` format. For example `06:00:00`.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(exportTimeRegex, "must be in HH:MM:SS format, e.g. 06:00:00"),
				},
			},
			"status": schema.StringAttribute{
				Description: "Report status. Valid values are `active` and `disabled`. Defaults to `active`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("active"),
				Validators: []validator.String{
					stringvalidator.OneOf("active", "disabled"),
				},
			},
			"columns": schema.ListAttribute{
				Description: "Columns to include in the exported report. If omitted, default columns are used.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"dsl_filter": schema.StringAttribute{
				Description: "Filter applied to the report data, as a JSON-encoded string. " +
					"Required for `alerts` and `compliance` report types. " +
					"Structure: `{\"filter\": [{\"field\": \"...\", \"includes\": [...]}]}`.",
				Optional: true,
			},
			"sonar_query": schema.StringAttribute{
				Description: "Discovery query as a JSON-encoded string. Required for `discovery` report types.",
				Optional:    true,
			},
			"sonar_query_params": schema.StringAttribute{
				Description: "Extra parameters for the discovery query as a JSON-encoded string, " +
					"e.g. `{\"additionalModels[]\": [\"CloudAccount\"], \"order_by[]\": [\"-OrcaScore\"], \"group_by[]\": [\"AlertType\"]}`.",
				Optional: true,
			},
			"query_filters": schema.StringAttribute{
				Description: "Extra query filters as a JSON-encoded string, e.g. `{\"show_informational_alerts\": true}`.",
				Optional:    true,
			},
			"config": schema.StringAttribute{
				Description: "Extra report configuration as a JSON-encoded string, e.g. compliance framework or compression settings.",
				Optional:    true,
			},
			"s3_path": schema.StringAttribute{
				Description: "Custom S3 path for the generated reports.",
				Optional:    true,
			},
			"recipients_emails": schema.ListAttribute{
				Description: "Email addresses to deliver the report to.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"custom_email_subject": schema.StringAttribute{
				Description: "Custom subject for report delivery emails.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(256),
				},
			},
			"custom_email_content": schema.StringAttribute{
				Description: "Custom body for report delivery emails.",
				Optional:    true,
			},
			"share_to_slack": schema.BoolAttribute{
				Description: "Deliver the report to a Slack channel. Requires `slack_channel`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"slack_channel": schema.SingleNestedAttribute{
				Description: "Slack delivery destination. Required when `share_to_slack` is `true`.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"workspace": schema.StringAttribute{
						Description: "Slack workspace name. Must be a connected Slack account in Orca.",
						Required:    true,
					},
					"channel": schema.StringAttribute{
						Description: "Slack channel ID. The Orca Slack app must be a member of this channel.",
						Required:    true,
					},
				},
			},
			"share_to_bucket": schema.BoolAttribute{
				Description: "Upload the report to an AWS S3 bucket. Requires `bucket`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"bucket": schema.StringAttribute{
				Description: "Name of the connected S3 bucket template. Required when `share_to_bucket` is `true`.",
				Optional:    true,
			},
			"share_to_azure_blob": schema.BoolAttribute{
				Description: "Upload the report to an Azure Blob Storage container. Requires `azure_blob_container`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"azure_blob_container": schema.StringAttribute{
				Description: "Name of the connected Azure Blob Storage container. Required when `share_to_azure_blob` is `true`.",
				Optional:    true,
			},
			"share_to_google_cloud_storage": schema.BoolAttribute{
				Description: "Upload the report to a Google Cloud Storage bucket. Requires `google_cloud_storage_template`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"google_cloud_storage_template": schema.StringAttribute{
				Description: "Name of the connected Google Cloud Storage template. Required when `share_to_google_cloud_storage` is `true`.",
				Optional:    true,
			},
			"share_to_snowflake": schema.BoolAttribute{
				Description: "Deliver the report to Snowflake. Only supported for the `csv` format. Requires `snowflake_template`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"snowflake_template": schema.StringAttribute{
				Description: "Name of the connected Snowflake template. Required when `share_to_snowflake` is `true`.",
				Optional:    true,
			},
		},
	}
}

// jsonAttributeToMap parses a JSON-encoded string attribute into a map.
func jsonAttributeToMap(value types.String, attribute string, diagnostics *jsonDiagnostics) map[string]interface{} {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil
	}
	result := make(map[string]interface{})
	if err := json.Unmarshal([]byte(value.ValueString()), &result); err != nil {
		diagnostics.AddError(attribute, err)
		return nil
	}
	return result
}

type jsonDiagnostics struct {
	errors []string
}

func (d *jsonDiagnostics) AddError(attribute string, err error) {
	d.errors = append(d.errors, fmt.Sprintf("attribute %q does not contain valid JSON: %s", attribute, err.Error()))
}

func stringList(ctx context.Context, value types.List) []string {
	result := []string{}
	if value.IsNull() || value.IsUnknown() {
		return result
	}
	_ = value.ElementsAs(ctx, &result, false)
	return result
}

// refreshJSONAttribute keeps the configured JSON string when present in state,
// otherwise (e.g. right after import) re-creates it from the API value.
func refreshJSONAttribute(state types.String, apiValue map[string]interface{}, diagnostics *diag.Diagnostics) types.String {
	if !state.IsNull() {
		return state
	}
	if len(apiValue) == 0 {
		return state
	}
	encoded, err := json.Marshal(apiValue)
	if err != nil {
		diagnostics.AddError("Error reading scheduled report", "Could not encode API value as JSON: "+err.Error())
		return state
	}
	return types.StringValue(string(encoded))
}

// stringOrNull returns the API value, or keeps the state null when the API
// returns an empty string for an attribute that was never configured.
func stringOrNull(apiValue string, state types.String) types.String {
	if apiValue == "" && state.IsNull() {
		return state
	}
	return types.StringValue(apiValue)
}

func (r *scheduledReportResource) buildAPIPayload(ctx context.Context, plan *scheduledReportResourceModel) (*api_client.ScheduledReport, []string) {
	jsonDiags := jsonDiagnostics{}

	status := reportStatusToInt[plan.Status.ValueString()]
	payload := api_client.ScheduledReport{
		Name:            plan.Name.ValueString(),
		Type:            plan.Type.ValueString(),
		Format:          plan.Format.ValueString(),
		Recurrence:      plan.Recurrence.ValueString(),
		FirstReportDate: plan.FirstReportDate.ValueString(),
		ExportTime:      plan.ExportTime.ValueString(),
		Status:          &status,

		Columns:          stringList(ctx, plan.Columns),
		DSLFilter:        jsonAttributeToMap(plan.DSLFilter, "dsl_filter", &jsonDiags),
		SonarQuery:       plan.SonarQuery.ValueString(),
		SonarQueryParams: jsonAttributeToMap(plan.SonarQueryParams, "sonar_query_params", &jsonDiags),
		QueryFilters:     jsonAttributeToMap(plan.QueryFilters, "query_filters", &jsonDiags),
		Config:           jsonAttributeToMap(plan.Config, "config", &jsonDiags),
		S3Path:           plan.S3Path.ValueString(),

		RecipientsEmails:   stringList(ctx, plan.RecipientsEmails),
		CustomEmailSubject: plan.CustomEmailSubject.ValueString(),
		CustomEmailContent: plan.CustomEmailContent.ValueString(),

		ShareToSlack: plan.ShareToSlack.ValueBool(),

		ShareToBucket: plan.ShareToBucket.ValueBool(),
		Bucket:        plan.Bucket.ValueString(),

		ShareToAzureBlob:   plan.ShareToAzureBlob.ValueBool(),
		AzureBlobContainer: plan.AzureBlobContainer.ValueString(),

		ShareToGoogleCloudStorage:  plan.ShareToGoogleCloudStorage.ValueBool(),
		GoogleCloudStorageTemplate: plan.GoogleCloudStorageTemplate.ValueString(),

		ShareToSnowflake:  plan.ShareToSnowflake.ValueBool(),
		SnowflakeTemplate: plan.SnowflakeTemplate.ValueString(),
	}

	if plan.SlackChannel != nil {
		payload.SlackChannel = map[string]interface{}{
			"workspace": plan.SlackChannel.Workspace.ValueString(),
			"channel":   plan.SlackChannel.Channel.ValueString(),
		}
	}

	return &payload, jsonDiags.errors
}

func (r *scheduledReportResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scheduledReportResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, jsonErrors := r.buildAPIPayload(ctx, &plan)
	for _, message := range jsonErrors {
		resp.Diagnostics.AddError("Error creating scheduled report", message)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.CreateScheduledReport(*payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating scheduled report",
			"Could not create scheduled report, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.Status = types.StringValue(reportStatusToString(instance.Status))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *scheduledReportResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scheduledReportResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.GetScheduledReport(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading scheduled report",
			fmt.Sprintf("Could not read scheduled report ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("Scheduled report %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(instance.Name)
	state.Type = types.StringValue(instance.Type)
	state.Format = types.StringValue(instance.Format)
	state.Recurrence = types.StringValue(instance.Recurrence)
	state.Status = types.StringValue(reportStatusToString(instance.Status))

	if state.FirstReportDate.IsNull() {
		state.FirstReportDate = types.StringValue(instance.FirstReportDate)
	}
	if state.ExportTime.IsNull() {
		state.ExportTime = types.StringValue(instance.ExportTime)
	}

	if len(instance.Columns) > 0 || !state.Columns.IsNull() {
		columns, columnsDiags := types.ListValueFrom(ctx, types.StringType, instance.Columns)
		resp.Diagnostics.Append(columnsDiags...)
		state.Columns = columns
	}
	if len(instance.RecipientsEmails) > 0 || !state.RecipientsEmails.IsNull() {
		emails, emailsDiags := types.ListValueFrom(ctx, types.StringType, instance.RecipientsEmails)
		resp.Diagnostics.Append(emailsDiags...)
		state.RecipientsEmails = emails
	}

	// JSON-encoded attributes are kept as configured to avoid spurious diffs
	// caused by server-side normalization. They are only refreshed from the
	// API when missing from state (e.g. on import).
	state.DSLFilter = refreshJSONAttribute(state.DSLFilter, instance.DSLFilter, &resp.Diagnostics)
	state.SonarQueryParams = refreshJSONAttribute(state.SonarQueryParams, instance.SonarQueryParams, &resp.Diagnostics)
	state.QueryFilters = refreshJSONAttribute(state.QueryFilters, instance.QueryFilters, &resp.Diagnostics)
	state.Config = refreshJSONAttribute(state.Config, instance.Config, &resp.Diagnostics)
	if state.SonarQuery.IsNull() && instance.SonarQuery != "" {
		state.SonarQuery = types.StringValue(instance.SonarQuery)
	}

	state.CustomEmailSubject = stringOrNull(instance.CustomEmailSubject, state.CustomEmailSubject)
	state.CustomEmailContent = stringOrNull(instance.CustomEmailContent, state.CustomEmailContent)
	state.S3Path = stringOrNull(instance.S3Path, state.S3Path)

	state.ShareToSlack = types.BoolValue(instance.ShareToSlack)
	if len(instance.SlackChannel) > 0 {
		state.SlackChannel = &slackChannelResourceModel{
			Workspace: types.StringValue(fmt.Sprintf("%v", instance.SlackChannel["workspace"])),
			Channel:   types.StringValue(fmt.Sprintf("%v", instance.SlackChannel["channel"])),
		}
	} else {
		state.SlackChannel = nil
	}

	state.ShareToBucket = types.BoolValue(instance.ShareToBucket)
	state.Bucket = stringOrNull(instance.Bucket, state.Bucket)

	state.ShareToAzureBlob = types.BoolValue(instance.ShareToAzureBlob)
	state.AzureBlobContainer = stringOrNull(instance.AzureBlobContainer, state.AzureBlobContainer)

	state.ShareToGoogleCloudStorage = types.BoolValue(instance.ShareToGoogleCloudStorage)
	state.GoogleCloudStorageTemplate = stringOrNull(instance.GoogleCloudStorageTemplate, state.GoogleCloudStorageTemplate)

	state.ShareToSnowflake = types.BoolValue(instance.ShareToSnowflake)
	state.SnowflakeTemplate = stringOrNull(instance.SnowflakeTemplate, state.SnowflakeTemplate)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *scheduledReportResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan scheduledReportResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, jsonErrors := r.buildAPIPayload(ctx, &plan)
	for _, message := range jsonErrors {
		resp.Diagnostics.AddError("Error updating scheduled report", message)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.UpdateScheduledReport(plan.ID.ValueString(), *payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating scheduled report",
			"Could not update scheduled report, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Status = types.StringValue(reportStatusToString(instance.Status))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *scheduledReportResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scheduledReportResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteScheduledReport(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting scheduled report",
			fmt.Sprintf("Could not delete scheduled report with ID %s, unexpected error: %s", state.ID.ValueString(), err.Error()),
		)
	}
}
