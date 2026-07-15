package zscaler

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type state struct {
	cc.CommonFields
	VanityDomain types.String `tfsdk:"vanity_domain"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

// buildPayload converts the planned state into the Zscaler API payload.
func buildPayload(_ context.Context, st cc.State, _ *diag.Diagnostics) api_client.ZscalerExternalServiceConfig {
	s := st.(*state)
	return api_client.ZscalerExternalServiceConfig{
		TemplateName: s.TemplateName.ValueString(),
		IsEnabled:    s.IsEnabled.ValueBool(),
		IsDefault:    s.IsDefault.ValueBool(),
		Config: api_client.ZscalerConfig{
			VanityDomain: s.VanityDomain.ValueString(),
			ClientID:     s.ClientID.ValueString(),
			ClientSecret: s.ClientSecret.ValueString(),
		},
	}
}

// extract maps the API envelope back onto state; an empty vanity_domain never clobbers the plan.
func extract(o *api_client.ZscalerExternalServiceConfig, st cc.State, _ *diag.Diagnostics) cc.APIObject {
	if o.Config.VanityDomain != "" {
		st.(*state).VanityDomain = types.StringValue(o.Config.VanityDomain)
	}
	return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault}
}

func NewZscalerResource() resource.Resource {
	return cc.New(cc.Spec[api_client.ZscalerExternalServiceConfig]{
		TypeNameSuffix: "_integration_zscaler_zpa",
		UIName:         "Zscaler ZPA integration",
		Description:    "Manage a Zscaler ZPA integration in Orca. Creates an external service config of `service_name = \"zscaler\"`. The OAuth `client_id` and `client_secret` are stored in Orca's secret store and are never returned by the API.",
		VariantAttributes: map[string]schema.Attribute{
			"vanity_domain": schema.StringAttribute{
				Required:    true,
				Description: "Zscaler ZPA vanity domain (the customer-specific tenant identifier used in the OAuth URL).",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"client_id": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Zscaler ZPA OAuth `client_id`. Stored in Orca's secret store; never returned by the API.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"client_secret": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Zscaler ZPA OAuth `client_secret`. Stored in Orca's secret store; never returned by the API.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
		NewState:     func() cc.State { return &state{} },
		BuildPayload: buildPayload,
		Extract:      extract,
		Create:       (*api_client.APIClient).CreateZscalerConfig,
		Get:          (*api_client.APIClient).GetZscalerConfig,
		Update:       (*api_client.APIClient).UpdateZscalerConfig,
		Delete:       (*api_client.APIClient).DeleteZscalerConfig,
	})
}
