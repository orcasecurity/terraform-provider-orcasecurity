package servicenow_itsm_template

import (
	"context"
	"encoding/json"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &templateResource{}
	_ resource.ResourceWithConfigure   = &templateResource{}
	_ resource.ResourceWithImportState = &templateResource{}
)

type templateResource struct {
	apiClient *api_client.APIClient
}

type templateResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	TemplateName             types.String `tfsdk:"template_name"`
	ResourceID               types.String `tfsdk:"resource_id"`
	InstanceName             types.String `tfsdk:"instance_name"`
	BaseURL                  types.String `tfsdk:"base_url"`
	Username                 types.String `tfsdk:"username"`
	ResolutionStatus         types.String `tfsdk:"resolution_status"`
	ResolutionCode           types.String `tfsdk:"resolution_code"`
	ResolutionNote           types.String `tfsdk:"resolution_note"`
	ReopenStatus             types.String `tfsdk:"reopen_status"`
	MappingJSON              types.String `tfsdk:"mapping_json"`
	OnCloseAlertMappingJSON  types.String `tfsdk:"on_close_alert_mapping_json"`
	AllowReopenAndResolution types.Bool   `tfsdk:"allow_reopen_and_resolution"`
	AllowMapping             types.Bool   `tfsdk:"allow_mapping"`
	IsEnabled                types.Bool   `tfsdk:"is_enabled"`
	IsDefault                types.Bool   `tfsdk:"is_default"`
}

func NewServiceNowITSMTemplateResource() resource.Resource {
	return &templateResource{}
}

func (r *templateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_servicenow_itsm_template"
}

func (r *templateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *templateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage a ServiceNow ITSM template in Orca. This creates an `external_service/config` row under `service_name = \"sn_incidents\"` with `config.type = \"ITSM\"` and links it to the credentials side of the integration (an `orcasecurity_integration_servicenow_itsm` resource). Templates carry the per-ticket settings — field mapping, resolution codes, reopen behaviour.",
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
				Optional:    true,
				Description: "ID of the `orcasecurity_integration_servicenow_itsm` resource that carries the ServiceNow credentials. Use the resource link when you want credentials managed separately from the template.",
			},
			"instance_name": schema.StringAttribute{
				Optional:    true,
				Description: "ServiceNow instance subdomain (for example, `ven03666`). Mutually exclusive with `base_url`. Required when `resource_id` is not set.",
			},
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "Full ServiceNow base URL (`https://...`). Mutually exclusive with `instance_name`. Takes precedence over `instance_name` if both are provided to the API.",
			},
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "Optional ServiceNow username override. Usually supplied via the linked resource.",
			},
			"resolution_status": schema.StringAttribute{
				Optional:    true,
				Description: "ServiceNow incident state code used when Orca resolves a ticket (for example, `6`).",
			},
			"resolution_code": schema.StringAttribute{
				Optional:    true,
				Description: "ServiceNow close code applied when Orca resolves a ticket.",
			},
			"resolution_note": schema.StringAttribute{
				Optional:    true,
				Description: "ServiceNow close notes applied when Orca resolves a ticket.",
			},
			"reopen_status": schema.StringAttribute{
				Optional:    true,
				Description: "ServiceNow incident state code Orca moves a ticket to when re-opening it.",
			},
			"mapping_json": schema.StringAttribute{
				Optional:    true,
				Description: "JSON-encoded `mapping` object that defines how Orca alert fields map to ServiceNow incident fields. Free-form per the Orca schema — each key is a ServiceNow field name and the value is a list of `{ \"orca\": \"<alert_field>\" }` or `{ \"value\": \"<literal>\" }` entries.",
			},
			"on_close_alert_mapping_json": schema.StringAttribute{
				Optional:    true,
				Description: "JSON-encoded `on_close_alert_mapping` object used when an Orca-driven close event syncs back to ServiceNow.",
			},
			"allow_reopen_and_resolution": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"allow_mapping": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"is_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the template is enabled. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"is_default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this template is the organisation's default ITSM template. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func decodeJSONField(s types.String, fieldName string) (json.RawMessage, diag.Diagnostics) {
	var diags diag.Diagnostics
	if s.IsNull() || s.IsUnknown() || s.ValueString() == "" {
		return nil, diags
	}
	raw := json.RawMessage(s.ValueString())
	// Validate the input is well-formed JSON; otherwise Orca returns an opaque 400.
	if !json.Valid(raw) {
		diags.AddError(fmt.Sprintf("Invalid JSON in %s", fieldName), "Value must be a JSON-encoded object.")
		return nil, diags
	}
	return raw, diags
}

func (r *templateResource) buildPayload(ctx context.Context, plan templateResourceModel, diags *diag.Diagnostics) api_client.ServiceNowITSMTemplate {
	allowReopen := plan.AllowReopenAndResolution.ValueBool()
	allowMapping := plan.AllowMapping.ValueBool()

	cfg := api_client.ServiceNowITSMTemplateConfig{
		Type:                     "ITSM",
		InstanceName:             plan.InstanceName.ValueString(),
		BaseURL:                  plan.BaseURL.ValueString(),
		Username:                 plan.Username.ValueString(),
		ResolutionStatus:         plan.ResolutionStatus.ValueString(),
		ResolutionCode:           plan.ResolutionCode.ValueString(),
		ResolutionNote:           plan.ResolutionNote.ValueString(),
		ReopenStatus:             plan.ReopenStatus.ValueString(),
		AllowReopenAndResolution: &allowReopen,
		AllowMapping:             &allowMapping,
	}

	mapping, mappingDiags := decodeJSONField(plan.MappingJSON, "mapping_json")
	diags.Append(mappingDiags...)
	cfg.Mapping = mapping

	onClose, onCloseDiags := decodeJSONField(plan.OnCloseAlertMappingJSON, "on_close_alert_mapping_json")
	diags.Append(onCloseDiags...)
	cfg.OnCloseAlertMapping = onClose

	payload := api_client.ServiceNowITSMTemplate{
		TemplateName: plan.TemplateName.ValueString(),
		Resource:     plan.ResourceID.ValueString(),
		IsEnabled:    plan.IsEnabled.ValueBool(),
		IsDefault:    plan.IsDefault.ValueBool(),
		Config:       cfg,
	}

	return payload
}

func encodeJSONField(raw json.RawMessage, planned types.String) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(raw) == 0 {
		if planned.IsNull() || planned.IsUnknown() {
			return types.StringNull(), diags
		}
		// Preserve the user-supplied empty/null distinction (an empty string in HCL stays as an
		// empty string in state).
		return planned, diags
	}
	// Normalise into the same shape the user supplied (compact JSON) so plans don't drift on
	// whitespace differences between the API response and the user's HCL.
	var generic interface{}
	if err := json.Unmarshal(raw, &generic); err != nil {
		diags.AddError("Invalid JSON from API", err.Error())
		return planned, diags
	}
	encoded, err := json.Marshal(generic)
	if err != nil {
		diags.AddError("Could not re-marshal JSON from API", err.Error())
		return planned, diags
	}
	return types.StringValue(string(encoded)), diags
}

func (r *templateResource) applyAPIResponse(ctx context.Context, plan *templateResourceModel, apiObj *api_client.ServiceNowITSMTemplate, diags *diag.Diagnostics) {
	plan.ID = types.StringValue(apiObj.ID)
	plan.IsEnabled = types.BoolValue(apiObj.IsEnabled)
	plan.IsDefault = types.BoolValue(apiObj.IsDefault)
	if apiObj.TemplateName != "" {
		plan.TemplateName = types.StringValue(apiObj.TemplateName)
	}
	if apiObj.Resource != "" {
		plan.ResourceID = types.StringValue(apiObj.Resource)
	}

	if apiObj.Config.InstanceName != "" {
		plan.InstanceName = types.StringValue(apiObj.Config.InstanceName)
	}
	if apiObj.Config.BaseURL != "" {
		plan.BaseURL = types.StringValue(apiObj.Config.BaseURL)
	}
	if apiObj.Config.Username != "" {
		plan.Username = types.StringValue(apiObj.Config.Username)
	}
	if apiObj.Config.ResolutionStatus != "" {
		plan.ResolutionStatus = types.StringValue(apiObj.Config.ResolutionStatus)
	}
	if apiObj.Config.ResolutionCode != "" {
		plan.ResolutionCode = types.StringValue(apiObj.Config.ResolutionCode)
	}
	if apiObj.Config.ResolutionNote != "" {
		plan.ResolutionNote = types.StringValue(apiObj.Config.ResolutionNote)
	}
	if apiObj.Config.ReopenStatus != "" {
		plan.ReopenStatus = types.StringValue(apiObj.Config.ReopenStatus)
	}
	if apiObj.Config.AllowReopenAndResolution != nil {
		plan.AllowReopenAndResolution = types.BoolValue(*apiObj.Config.AllowReopenAndResolution)
	}
	if apiObj.Config.AllowMapping != nil {
		plan.AllowMapping = types.BoolValue(*apiObj.Config.AllowMapping)
	}

	mapping, mappingDiags := encodeJSONField(apiObj.Config.Mapping, plan.MappingJSON)
	diags.Append(mappingDiags...)
	plan.MappingJSON = mapping

	onClose, onCloseDiags := encodeJSONField(apiObj.Config.OnCloseAlertMapping, plan.OnCloseAlertMappingJSON)
	diags.Append(onCloseDiags...)
	plan.OnCloseAlertMappingJSON = onClose
}

func (r *templateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating ServiceNow ITSM template", "API client not configured.")
		return
	}

	var plan templateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := r.buildPayload(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.apiClient.CreateServiceNowITSMTemplate(payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating ServiceNow ITSM template",
			fmt.Sprintf("Could not create ServiceNow ITSM template: %s", err.Error()),
		)
		return
	}

	r.applyAPIResponse(ctx, &plan, created, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *templateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state templateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.apiClient.GetServiceNowITSMTemplate(state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading ServiceNow ITSM template",
			fmt.Sprintf("Could not read ServiceNow ITSM template %s: %s", state.TemplateName.ValueString(), err.Error()),
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

func (r *templateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan templateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state templateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := r.buildPayload(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.apiClient.UpdateServiceNowITSMTemplate(state.TemplateName.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating ServiceNow ITSM template",
			fmt.Sprintf("Could not update ServiceNow ITSM template %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	r.applyAPIResponse(ctx, &plan, updated, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *templateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state templateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteServiceNowITSMTemplate(state.TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting ServiceNow ITSM template",
			fmt.Sprintf("Could not delete ServiceNow ITSM template %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *templateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
