package slack

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mappingElemType is the element type of the `mapping` attribute: each section
// (e.g. "title", "description") maps to an ordered list of Orca alert field names.
var mappingElemType = types.ListType{ElemType: types.StringType}

// state is the Slack template Terraform model. CommonFieldsWithBU carries id / template_name /
// is_enabled / is_default / business_units plus the GetCommon/SetCommon glue the generic spec
// needs. Slack has no linked OAuth resource, so there is no resource_id.
type state struct {
	cc.CommonFieldsWithBU
	WorkspaceID types.String `tfsdk:"workspace_id"`
	Channels    types.List   `tfsdk:"channels"`
	ShowActions types.Bool   `tfsdk:"show_actions"`
	Mapping     types.Map    `tfsdk:"mapping"`
}

func variantAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"workspace_id": schema.StringAttribute{
			Required:    true,
			Description: "Slack workspace (team) ID Orca posts to, e.g. `T0A0KSCQ1B3`. The workspace must already be connected to Orca via the Slack app.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"channels": schema.ListAttribute{
			Required:    true,
			ElementType: types.StringType,
			Description: "Ordered list of Slack channel IDs Orca sends alerts to, e.g. `[\"C0AE82CGDH7\"]`.",
		},
		"show_actions": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(true),
			Description: "Whether Slack messages include interactive action buttons. Default `true`.",
		},
		"mapping": schema.MapAttribute{
			Required:    true,
			ElementType: mappingElemType,
			Description: "Map of message section name (e.g. `title`, `description`) to an ordered list of Orca alert field names rendered in that section. Field names are wrapped as `{\"orca\": \"<field>\"}` on the wire.",
		},
		// Override the base business_units attribute: Orca does not accept BU changes on update
		// for slack, so a change forces replace.
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

// decodeMapping converts the friendly `mapping` attribute (section -> list of Orca field
// names) into the wire shape `{ "<section>": [ { "orca": "<field>" }, ... ] }`.
func decodeMapping(ctx context.Context, s *state, cfg *api_client.SlackConfig, diags *diag.Diagnostics) {
	if s.Mapping.IsNull() || s.Mapping.IsUnknown() {
		return
	}
	var sections map[string][]string
	diags.Append(s.Mapping.ElementsAs(ctx, &sections, false)...)
	if diags.HasError() {
		return
	}
	wire := make(map[string][]map[string]string, len(sections))
	for section, fields := range sections {
		entries := make([]map[string]string, 0, len(fields))
		for _, f := range fields {
			entries = append(entries, map[string]string{"orca": f})
		}
		wire[section] = entries
	}
	raw, err := json.Marshal(wire)
	if err != nil {
		diags.AddError("Failed to encode Slack mapping", err.Error())
		return
	}
	cfg.Mapping = raw
}

// encodeMapping converts the wire mapping from the API response back into the friendly
// section -> list-of-field-names shape. Any entry that is not a single {"orca": ...}
// reference surfaces an error rather than being silently dropped.
func encodeMapping(ctx context.Context, s *state, cfg *api_client.SlackConfig, diags *diag.Diagnostics) {
	if len(cfg.Mapping) == 0 {
		return
	}
	var wire map[string][]map[string]string
	if err := json.Unmarshal(cfg.Mapping, &wire); err != nil {
		diags.AddError("Failed to decode Slack mapping from API", err.Error())
		return
	}
	sections := make(map[string][]string, len(wire))
	for section, entries := range wire {
		fields := make([]string, 0, len(entries))
		for _, e := range entries {
			orca, ok := e["orca"]
			if !ok || len(e) != 1 {
				diags.AddError(
					"Unsupported Slack mapping entry",
					"section \""+section+"\" contains a mapping entry that is not a single {\"orca\": ...} reference. "+
						"This resource only supports Orca-field mappings.",
				)
				return
			}
			fields = append(fields, orca)
		}
		sections[section] = fields
	}
	m, d := types.MapValueFrom(ctx, mappingElemType, sections)
	diags.Append(d...)
	if !diags.HasError() {
		s.Mapping = m
	}
}

func NewSlackResource() resource.Resource {
	return cc.New(cc.Spec[api_client.SlackTemplate]{
		TypeNameSuffix:        "_integration_slack_template",
		UIName:                "Slack template",
		Description:           "Manage a Slack integration in Orca. Creates an external service config of `service_name = \"slack\"` that posts Orca alerts to the given Slack workspace/channels, with a field mapping controlling which alert fields render in the message title and description.",
		SupportsBusinessUnits: true,
		VariantAttributes:     variantAttributes(),
		NewState:              func() cc.State { return &state{} },
		BuildPayload: func(ctx context.Context, st cc.State, diags *diag.Diagnostics) api_client.SlackTemplate {
			s := st.(*state)
			channels, d := common.StringSliceFromList(ctx, s.Channels)
			diags.Append(d...)
			cfg := api_client.SlackConfig{
				WorkspaceID: s.WorkspaceID.ValueString(),
				Channels:    channels,
				ShowActions: s.ShowActions.ValueBool(),
			}
			decodeMapping(ctx, s, &cfg, diags)
			return api_client.SlackTemplate{
				TemplateName:  s.TemplateName.ValueString(),
				IsEnabled:     s.IsEnabled.ValueBool(),
				IsDefault:     s.IsDefault.ValueBool(),
				Config:        cfg,
				BusinessUnits: common.BusinessUnitsToAPI(ctx, s.BusinessUnits, diags),
			}
		},
		Extract: func(o *api_client.SlackTemplate, st cc.State, diags *diag.Diagnostics) cc.APIObject {
			s := st.(*state)
			if o.Config.WorkspaceID != "" {
				s.WorkspaceID = types.StringValue(o.Config.WorkspaceID)
			}
			channels, d := common.OptionalListMatchPlan(context.Background(), s.Channels, o.Config.Channels)
			diags.Append(d...)
			s.Channels = channels
			s.ShowActions = types.BoolValue(o.Config.ShowActions)
			encodeMapping(context.Background(), s, &o.Config, diags)
			return cc.APIObject{
				ID:            o.ID,
				TemplateName:  o.TemplateName,
				IsEnabled:     o.IsEnabled,
				IsDefault:     o.IsDefault,
				BusinessUnits: o.BusinessUnits,
			}
		},
		Create: (*api_client.APIClient).CreateSlackTemplate,
		Get:    (*api_client.APIClient).GetSlackTemplate,
		Update: (*api_client.APIClient).UpdateSlackTemplate,
		Delete: (*api_client.APIClient).DeleteSlackTemplate,
	})
}
