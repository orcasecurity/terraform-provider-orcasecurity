package monday_template

import (
	"context"
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

// state is the Monday template Terraform model. CommonFieldsWithBU carries id / template_name /
// is_enabled / is_default / business_units plus the GetCommon/SetCommon glue the generic spec
// needs. resource_id maps to the envelope's top-level "resource".
type state struct {
	cc.CommonFieldsWithBU
	ResourceID              types.String `tfsdk:"resource_id"`
	WorkspaceID             types.String `tfsdk:"workspace_id"`
	BoardID                 types.String `tfsdk:"board_id"`
	GroupID                 types.String `tfsdk:"group_id"`
	MappingJSON             types.String `tfsdk:"mapping_json"`
	AlertStatusMappingJSON  types.String `tfsdk:"alert_status_mapping_json"`
	TicketStatusMappingJSON types.String `tfsdk:"ticket_status_mapping_json"`
}

func variantAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"resource_id": schema.StringAttribute{
			Required:    true,
			Description: "UUID of the Monday resource that carries the credentials (look it up in the Orca UI under Integrations → Monday.com).",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"board_id": schema.StringAttribute{
			Required:    true,
			Description: "Monday board ID Orca opens items in.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"workspace_id": schema.StringAttribute{
			Optional:    true,
			Description: "Monday workspace ID that owns the board.",
		},
		"group_id": schema.StringAttribute{
			Optional:    true,
			Description: "Monday group (section) ID within the board where new items are created.",
		},
		"mapping_json": schema.StringAttribute{
			Optional:    true,
			Description: "JSON-encoded `mapping` object. Each key is a Monday column ID; values are lists of `{ \"orca\": \"<alert_field>\" }`, a `{ \"custom\": \"<literal>\" }` object, a `{ \"value\": \"<literal>\" }` object, or a list of `{ \"value\": { \"id\": ..., \"kind\": ... } }` entries for people columns.",
		},
		"alert_status_mapping_json": schema.StringAttribute{
			Optional:    true,
			Description: "JSON-encoded `alert_status_mapping` — maps Orca alert statuses to Monday status column values (for example, `{\"snoozed\": \"1\"}`).",
		},
		"ticket_status_mapping_json": schema.StringAttribute{
			Optional:    true,
			Description: "JSON-encoded `ticket_status_mapping` — maps Monday status column values back to Orca alert state changes (for example, `{\"2\": {\"status\": \"dismissed\"}}`).",
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

// decodeMappings pulls the three JSON-string fields off the plan into the API config.
func decodeMappings(s *state, cfg *api_client.MondayTemplateConfig, diags *diag.Diagnostics) {
	common.DecodeJSONFields([]common.JSONFieldDecode{
		{Src: s.MappingJSON, Field: "mapping_json", Dst: &cfg.Mapping},
		{Src: s.AlertStatusMappingJSON, Field: "alert_status_mapping_json", Dst: &cfg.AlertStatusMapping},
		{Src: s.TicketStatusMappingJSON, Field: "ticket_status_mapping_json", Dst: &cfg.TicketStatusMapping},
	}, diags)
}

// encodeMappings writes the three JSON config fields from the API response back onto state,
// preserving each field's planned whitespace shape via EncodeJSONField.
func encodeMappings(s *state, cfg *api_client.MondayTemplateConfig, diags *diag.Diagnostics) {
	common.EncodeJSONFields([]common.JSONFieldEncode{
		{Raw: cfg.Mapping, Dst: &s.MappingJSON},
		{Raw: cfg.AlertStatusMapping, Dst: &s.AlertStatusMappingJSON},
		{Raw: cfg.TicketStatusMapping, Dst: &s.TicketStatusMappingJSON},
	}, diags)
}

func NewMondayTemplateResource() resource.Resource {
	return cc.New(cc.Spec[api_client.MondayTemplate]{
		TypeNameSuffix:        "_integration_monday_template",
		UIName:                "Monday template",
		Description:           "Manage a Monday.com template in Orca. Creates an external service config of `service_name = \"monday\"` linked to an existing Monday resource. Holds the board, group, field-mapping, and status-mapping settings used when Orca opens Monday items.",
		SupportsBusinessUnits: true,
		VariantAttributes:     variantAttributes(),
		NewState:              func() cc.State { return &state{} },
		BuildPayload: func(ctx context.Context, st cc.State, diags *diag.Diagnostics) api_client.MondayTemplate {
			s := st.(*state)
			cfg := api_client.MondayTemplateConfig{
				WorkspaceID: s.WorkspaceID.ValueString(),
				BoardID:     s.BoardID.ValueString(),
				GroupID:     s.GroupID.ValueString(),
			}
			decodeMappings(s, &cfg, diags)
			return api_client.MondayTemplate{
				TemplateName:  s.TemplateName.ValueString(),
				Resource:      s.ResourceID.ValueString(),
				IsEnabled:     s.IsEnabled.ValueBool(),
				IsDefault:     s.IsDefault.ValueBool(),
				Config:        cfg,
				BusinessUnits: common.BusinessUnitsToAPI(ctx, s.BusinessUnits, diags),
			}
		},
		Extract: func(o *api_client.MondayTemplate, st cc.State, diags *diag.Diagnostics) cc.APIObject {
			s := st.(*state)
			if o.Resource != "" {
				s.ResourceID = types.StringValue(o.Resource)
			}
			if o.Config.WorkspaceID != "" {
				s.WorkspaceID = types.StringValue(o.Config.WorkspaceID)
			}
			if o.Config.BoardID != "" {
				s.BoardID = types.StringValue(o.Config.BoardID)
			}
			if o.Config.GroupID != "" {
				s.GroupID = types.StringValue(o.Config.GroupID)
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
		Create: (*api_client.APIClient).CreateMondayTemplate,
		Get:    (*api_client.APIClient).GetMondayTemplate,
		Update: (*api_client.APIClient).UpdateMondayTemplate,
		Delete: (*api_client.APIClient).DeleteMondayTemplate,
	})
}
