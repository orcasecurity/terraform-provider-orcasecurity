package webhook

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"
	wvc "terraform-provider-orcasecurity/orcasecurity/webhook_variant_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// webhookTypes are the variant discriminators the generic webhook resource accepts.
var webhookTypes = []string{"common", "torq", "tines", "opus", "coralogix", "panther"}

type webhookConfigModel struct {
	WebhookURL    types.String `tfsdk:"webhook_url"`
	Type          types.String `tfsdk:"type"`
	APIKey        types.String `tfsdk:"api_key"`
	BodyFields    types.List   `tfsdk:"body_fields"`
	CustomHeaders types.Map    `tfsdk:"custom_headers"`
}

// state is the generic-webhook TF model. It keeps the config fields under a nested `config`
// block (the original resource shape); CommonFieldsWithBU brings in the id / template_name /
// is_enabled / is_default / business_units quartet.
type state struct {
	cc.CommonFieldsWithBU
	Config *webhookConfigModel `tfsdk:"config"`
}

func NewWebhookResource() resource.Resource {
	return cc.New(cc.Spec[api_client.WebhookExternalServiceConfig]{
		TypeNameSuffix:        "_integration_webhook_template",
		UIName:                "Webhook integration",
		Description:           "Manage a Webhook integration in Orca. Creates an external service config of `service_name = \"webhook\"` so automations can fire HTTP callbacks to a customer-controlled endpoint.",
		SupportsBusinessUnits: true,
		VariantAttributes: map[string]schema.Attribute{
			"config": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Webhook configuration.",
				Attributes: map[string]schema.Attribute{
					"webhook_url": schema.StringAttribute{
						Required:    true,
						Description: "Destination URL Orca posts events to.",
						Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
					},
					"type": schema.StringAttribute{
						Required:    true,
						Description: "Webhook variant. One of: `common`, `torq`, `tines`, `opus`, `coralogix`, `panther`.",
						Validators:  []validator.String{stringvalidator.OneOf(webhookTypes...)},
					},
					"api_key": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Optional API key sent with each webhook request. Treated as sensitive.",
					},
					"body_fields": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Optional list of Orca alert fields to include in the webhook request body. Leave unset to send the default payload.",
						PlanModifiers: []planmodifier.List{
							wvc.EmptyListToNullModifier{},
						},
					},
					"custom_headers": schema.MapAttribute{
						Optional:    true,
						ElementType: wvc.CustomHeaderListType(),
						Description: "Optional custom HTTP headers, keyed by header name. Each value is a list of `{ custom = \"<value>\" }` objects so a single header can carry multiple values.",
						PlanModifiers: []planmodifier.Map{
							wvc.EmptyMapToNullModifier{},
						},
					},
				},
			},
		},
		NewState: func() cc.State { return &state{} },
		BuildPayload: func(ctx context.Context, st cc.State, diags *diag.Diagnostics) api_client.WebhookExternalServiceConfig {
			s := st.(*state)
			payload := api_client.WebhookExternalServiceConfig{
				TemplateName: s.TemplateName.ValueString(),
				IsEnabled:    s.IsEnabled.ValueBool(),
				IsDefault:    s.IsDefault.ValueBool(),
			}
			if s.Config != nil {
				payload.Config = api_client.WebhookResourceConfig{
					WebhookURL: s.Config.WebhookURL.ValueString(),
					Type:       s.Config.Type.ValueString(),
					APIKey:     s.Config.APIKey.ValueString(),
				}
				if !s.Config.BodyFields.IsNull() && !s.Config.BodyFields.IsUnknown() {
					var fields []string
					diags.Append(s.Config.BodyFields.ElementsAs(ctx, &fields, false)...)
					payload.Config.BodyFields = fields
				}
				headers, headerDiags := wvc.CustomHeadersToAPI(ctx, s.Config.CustomHeaders)
				diags.Append(headerDiags...)
				payload.Config.CustomHeaders = headers
			}
			payload.BusinessUnits = common.BusinessUnitsToAPI(ctx, s.BusinessUnits, diags)
			return payload
		},
		// Create/Update touch only top-level fields so the planned (sensitive) config survives the
		// Plugin Framework's consistency check — shared with the webhook variants.
		Extract: wvc.ExtractTopLevel,
		// Read refreshes the whole config so the next plan detects drift. api_key is intentionally
		// not overwritten from API echoes — the user-supplied secret already lives in state.
		ExtractOnRead: extractFull,
		Create:        (*api_client.APIClient).CreateWebhookConfig,
		Get:           (*api_client.APIClient).GetWebhookConfigByTemplate,
		Update:        (*api_client.APIClient).UpdateWebhookConfig,
		Delete:        (*api_client.APIClient).DeleteWebhookConfig,
	})
}

// extractFull is the Read-path refresh of the whole config block. Reuses the shared webhook
// conversion helpers so there is one source of truth for body_fields / custom_headers.
func extractFull(api *api_client.WebhookExternalServiceConfig, st cc.State, diags *diag.Diagnostics) cc.APIObject {
	s := st.(*state)
	if s.Config == nil {
		s.Config = &webhookConfigModel{}
	}
	s.Config.WebhookURL = types.StringValue(api.Config.WebhookURL)
	s.Config.Type = types.StringValue(api.Config.Type)
	if s.Config.APIKey.IsUnknown() {
		if api.Config.APIKey != "" {
			s.Config.APIKey = types.StringValue(api.Config.APIKey)
		} else {
			s.Config.APIKey = types.StringNull()
		}
	}
	bodyFields, bfDiags := wvc.BodyFieldsFromAPI(context.Background(), api.Config.BodyFields, s.Config.BodyFields)
	diags.Append(bfDiags...)
	s.Config.BodyFields = bodyFields
	headers, headerDiags := wvc.CustomHeadersFromAPI(api.Config.CustomHeaders, s.Config.CustomHeaders)
	diags.Append(headerDiags...)
	s.Config.CustomHeaders = headers
	return wvc.ExtractTopLevel(api, st, diags)
}
