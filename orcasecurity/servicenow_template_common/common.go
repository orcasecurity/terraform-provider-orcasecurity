// Package servicenow_template_common builds the Spec[ServiceNowITSMTemplate] that the ITSM
// and SIR template resources share. Both variants hit /api/external_service/config with
// service_name="sn_incidents" and only differ in config.type ("ITSM" vs "SIR"); collapsing
// them onto the generic Spec[P] in config_integration_common keeps the CRUD loop in exactly
// one place.
package servicenow_template_common

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// state is the per-variant Terraform model. CommonFields gives us id / template_name /
// is_enabled / is_default plus the GetCommon/SetCommon glue the generic spec needs.
type state struct {
	cc.CommonFields
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
}

// Options is what each variant package supplies. ConfigType pins the sn_incidents config.type
// discriminator ("ITSM" or "SIR"); the four CRUD method refs let us reuse the typed wrappers
// on *api_client.APIClient (CreateServiceNowITSMTemplate vs CreateServiceNowSIRTemplate).
type Options struct {
	TypeNameSuffix string
	UIName         string
	Description    string
	ConfigType     string
	Create         func(*api_client.APIClient, api_client.ServiceNowITSMTemplate) (*api_client.ServiceNowITSMTemplate, error)
	Get            func(*api_client.APIClient, string) (*api_client.ServiceNowITSMTemplate, error)
	Update         func(*api_client.APIClient, string, api_client.ServiceNowITSMTemplate) (*api_client.ServiceNowITSMTemplate, error)
	Delete         func(*api_client.APIClient, string) error
}

func variantAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
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
	}
}

// extractStateFromAPI copies non-empty API response fields into the Terraform state model.
// Keeping this in a named function drops the cognitive complexity of the Extract closure.
func extractStateFromAPI(api *api_client.ServiceNowITSMTemplate, s *state, diags *diag.Diagnostics) {
	if api.Resource != "" {
		s.ResourceID = types.StringValue(api.Resource)
	}
	if api.Config.InstanceName != "" {
		s.InstanceName = types.StringValue(api.Config.InstanceName)
	}
	if api.Config.BaseURL != "" {
		s.BaseURL = types.StringValue(api.Config.BaseURL)
	}
	if api.Config.Username != "" {
		s.Username = types.StringValue(api.Config.Username)
	}
	if api.Config.ResolutionStatus != "" {
		s.ResolutionStatus = types.StringValue(api.Config.ResolutionStatus)
	}
	if api.Config.ResolutionCode != "" {
		s.ResolutionCode = types.StringValue(api.Config.ResolutionCode)
	}
	if api.Config.ResolutionNote != "" {
		s.ResolutionNote = types.StringValue(api.Config.ResolutionNote)
	}
	if api.Config.ReopenStatus != "" {
		s.ReopenStatus = types.StringValue(api.Config.ReopenStatus)
	}
	if api.Config.AllowReopenAndResolution != nil {
		s.AllowReopenAndResolution = types.BoolValue(*api.Config.AllowReopenAndResolution)
	}
	if api.Config.AllowMapping != nil {
		s.AllowMapping = types.BoolValue(*api.Config.AllowMapping)
	}
	// JSON-field round-trip uses the EncodeJSONField helper so plans don't drift on
	// whitespace differences between the API response and the user's HCL.
	mapping, mDiags := common.EncodeJSONField(api.Config.Mapping, s.MappingJSON)
	diags.Append(mDiags...)
	s.MappingJSON = mapping
	onClose, ocDiags := common.EncodeJSONField(api.Config.OnCloseAlertMapping, s.OnCloseAlertMappingJSON)
	diags.Append(ocDiags...)
	s.OnCloseAlertMappingJSON = onClose
}

// NewResource returns a resource.Resource built on top of the generic Spec[P]. There is no
// per-variant CRUD code: the only delta between ITSM and SIR is opts.ConfigType plus the
// CRUD method refs (which pin service_name / config.type / GET filter inside api_client).
func NewResource(opts Options) resource.Resource {
	return cc.New(cc.Spec[api_client.ServiceNowITSMTemplate]{
		TypeNameSuffix:        opts.TypeNameSuffix,
		UIName:                opts.UIName,
		Description:           opts.Description,
		SupportsBusinessUnits: false,
		VariantAttributes:     variantAttributes(),
		NewState:              func() cc.State { return &state{} },
		BuildPayload: func(_ context.Context, st cc.State, diags *diag.Diagnostics) api_client.ServiceNowITSMTemplate {
			s := st.(*state)
			allowReopen := s.AllowReopenAndResolution.ValueBool()
			allowMapping := s.AllowMapping.ValueBool()

			cfg := api_client.ServiceNowITSMTemplateConfig{
				Type:                     opts.ConfigType,
				InstanceName:             s.InstanceName.ValueString(),
				BaseURL:                  s.BaseURL.ValueString(),
				Username:                 s.Username.ValueString(),
				ResolutionStatus:         s.ResolutionStatus.ValueString(),
				ResolutionCode:           s.ResolutionCode.ValueString(),
				ResolutionNote:           s.ResolutionNote.ValueString(),
				ReopenStatus:             s.ReopenStatus.ValueString(),
				AllowReopenAndResolution: &allowReopen,
				AllowMapping:             &allowMapping,
			}

			mapping, mDiags := common.DecodeJSONField(s.MappingJSON, "mapping_json")
			diags.Append(mDiags...)
			cfg.Mapping = mapping

			onClose, ocDiags := common.DecodeJSONField(s.OnCloseAlertMappingJSON, "on_close_alert_mapping_json")
			diags.Append(ocDiags...)
			cfg.OnCloseAlertMapping = onClose

			return api_client.ServiceNowITSMTemplate{
				TemplateName: s.TemplateName.ValueString(),
				Resource:     s.ResourceID.ValueString(),
				IsEnabled:    s.IsEnabled.ValueBool(),
				IsDefault:    s.IsDefault.ValueBool(),
				Config:       cfg,
			}
		},
		Extract: func(api *api_client.ServiceNowITSMTemplate, st cc.State, diags *diag.Diagnostics) cc.APIObject {
			s := st.(*state)
			extractStateFromAPI(api, s, diags)
			return cc.APIObject{
				ID:           api.ID,
				TemplateName: api.TemplateName,
				IsEnabled:    api.IsEnabled,
				IsDefault:    api.IsDefault,
			}
		},
		Create: opts.Create,
		Get:    opts.Get,
		Update: opts.Update,
		Delete: opts.Delete,
	})
}
