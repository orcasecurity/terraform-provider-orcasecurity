// Package servicenow_template_common holds the shared model and helpers behind the ITSM and
// SIR template resources. Both variants speak to the same external_service/config endpoint
// under service_name="sn_incidents" and only differ in config.type ("ITSM" vs "SIR"), so the
// per-variant resource files end up reimplementing the same Read/Create/Update plumbing
// otherwise. Extracting it here drops their duplication from ~70% to thin wrappers.
package servicenow_template_common

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ResourceModel is the Terraform Plugin Framework model for the ITSM/SIR template attributes.
// It's shared so the per-variant resource files can declare a single struct field
// (“servicenow_template_common.ResourceModel“) rather than duplicating ~20 attributes each.
type ResourceModel struct {
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

// Variant is what each per-variant package provides — the only delta between ITSM and SIR
// resources (other than naming). The CRUD method pointers let this package call the right
// API client function without taking a dependency on the per-variant constants.
type Variant struct {
	TypeNameSuffix string // for example "_integration_servicenow_itsm_template"
	UIName         string // for error messages ("ServiceNow ITSM template")
	Create         func(client *api_client.APIClient, payload api_client.ServiceNowITSMTemplate) (*api_client.ServiceNowITSMTemplate, error)
	Get            func(client *api_client.APIClient, templateName string) (*api_client.ServiceNowITSMTemplate, error)
	Update         func(client *api_client.APIClient, templateName string, payload api_client.ServiceNowITSMTemplate) (*api_client.ServiceNowITSMTemplate, error)
	Delete         func(client *api_client.APIClient, templateName string) error
}

// Schema returns the shared schema for both variants. The per-variant resource just embeds
// the returned “schema.Schema“ and overrides “Description“.
func Schema(description string) schema.Schema {
	return schema.Schema{
		Description: description,
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
				Description: "ID of the `orcasecurity_integration_servicenow` resource that carries the credentials.",
			},
			"instance_name": schema.StringAttribute{
				Optional:    true,
				Description: "ServiceNow instance subdomain. Mutually exclusive with `base_url`. Required when no `resource_id` is set.",
			},
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "Full ServiceNow base URL (`https://...`). Mutually exclusive with `instance_name`.",
			},
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "Optional ServiceNow username override. Usually inherited from the linked resource.",
			},
			"resolution_status": schema.StringAttribute{
				Optional:    true,
				Description: "ServiceNow state code Orca moves a record to when resolving it.",
			},
			"resolution_code": schema.StringAttribute{
				Optional:    true,
				Description: "Close code applied when Orca resolves a record.",
			},
			"resolution_note": schema.StringAttribute{
				Optional:    true,
				Description: "Close notes applied when Orca resolves a record.",
			},
			"reopen_status": schema.StringAttribute{
				Optional:    true,
				Description: "ServiceNow state code Orca moves a record to when re-opening it.",
			},
			"mapping_json": schema.StringAttribute{
				Optional:    true,
				Description: "JSON-encoded `mapping` object describing how Orca alert fields map to ServiceNow fields. Each key is a ServiceNow field; values are lists of `{ \"orca\": \"<alert_field>\" }` or `{ \"value\": \"<literal>\" }` entries.",
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
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"is_default": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
		},
	}
}

// BuildPayload converts a plan model into an API payload. “configType“ is the
// “config.type“ value that the per-variant resource pins ("ITSM" or "SIR").
func BuildPayload(plan ResourceModel, configType string, diags *diag.Diagnostics) api_client.ServiceNowITSMTemplate {
	allowReopen := plan.AllowReopenAndResolution.ValueBool()
	allowMapping := plan.AllowMapping.ValueBool()

	cfg := api_client.ServiceNowITSMTemplateConfig{
		Type:                     configType,
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

	mapping, mappingDiags := common.DecodeJSONField(plan.MappingJSON, "mapping_json")
	diags.Append(mappingDiags...)
	cfg.Mapping = mapping

	onClose, onCloseDiags := common.DecodeJSONField(plan.OnCloseAlertMappingJSON, "on_close_alert_mapping_json")
	diags.Append(onCloseDiags...)
	cfg.OnCloseAlertMapping = onClose

	return api_client.ServiceNowITSMTemplate{
		TemplateName: plan.TemplateName.ValueString(),
		Resource:     plan.ResourceID.ValueString(),
		IsEnabled:    plan.IsEnabled.ValueBool(),
		IsDefault:    plan.IsDefault.ValueBool(),
		Config:       cfg,
	}
}

// ApplyAPIResponse copies the API response back into the plan/state model. “plan“ is
// updated in place.
func ApplyAPIResponse(plan *ResourceModel, apiObj *api_client.ServiceNowITSMTemplate, diags *diag.Diagnostics) {
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

	mapping, mappingDiags := common.EncodeJSONField(apiObj.Config.Mapping, plan.MappingJSON)
	diags.Append(mappingDiags...)
	plan.MappingJSON = mapping

	onClose, onCloseDiags := common.EncodeJSONField(apiObj.Config.OnCloseAlertMapping, plan.OnCloseAlertMappingJSON)
	diags.Append(onCloseDiags...)
	plan.OnCloseAlertMappingJSON = onClose
}

// Resource is the embeddable base for ITSM/SIR template resources. Per-variant code only
// needs to declare a “Variant“ and call “Resource{Variant: v}.*“ for the CRUD methods.
type Resource struct {
	APIClient  *api_client.APIClient
	Variant    Variant
	ConfigType string // "ITSM" or "SIR"
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.Variant.TypeNameSuffix
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.APIClient = req.ProviderData.(*api_client.APIClient)
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.APIClient == nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error creating %s", r.Variant.UIName), "API client not configured.")
		return
	}

	var plan ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := BuildPayload(plan, r.ConfigType, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.Variant.Create(r.APIClient, payload)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error creating %s", r.Variant.UIName),
			fmt.Sprintf("Could not create %s: %s", r.Variant.UIName, err.Error()),
		)
		return
	}

	ApplyAPIResponse(&plan, created, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.Variant.Get(r.APIClient, state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading %s", r.Variant.UIName),
			fmt.Sprintf("Could not read %s %s: %s", r.Variant.UIName, state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
	if current == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	ApplyAPIResponse(&state, current, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := BuildPayload(plan, r.ConfigType, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.Variant.Update(r.APIClient, state.TemplateName.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating %s", r.Variant.UIName),
			fmt.Sprintf("Could not update %s %s: %s", r.Variant.UIName, state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	ApplyAPIResponse(&plan, updated, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.Variant.Delete(r.APIClient, state.TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting %s", r.Variant.UIName),
			fmt.Sprintf("Could not delete %s %s: %s", r.Variant.UIName, state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
