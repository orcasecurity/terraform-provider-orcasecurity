package akamai

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
	AccessToken  types.String `tfsdk:"access_token"`
	ClientToken  types.String `tfsdk:"client_token"`
	ClientSecret types.String `tfsdk:"client_secret"`
	Host         types.String `tfsdk:"host"`
}

func NewAkamaiResource() resource.Resource {
	return cc.New(cc.Spec[api_client.AkamaiExternalServiceConfig]{
		TypeNameSuffix: "_integration_akamai",
		UIName:         "Akamai integration",
		Description:    "Manage an Akamai integration in Orca. Creates an external service config of `service_name = \"akamai\"`. The Akamai credentials (`access_token`, `client_token`, `client_secret`) are stored in Orca's secret store and are never returned by the API.",
		VariantAttributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Akamai EdgeGrid `access_token`.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"client_token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Akamai EdgeGrid `client_token`.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"client_secret": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Akamai EdgeGrid `client_secret`.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"host": schema.StringAttribute{
				Required:    true,
				Description: "Akamai EdgeGrid host (for example, `akab-xxxxxxxx.luna.akamaiapis.net`).",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
		NewState: func() cc.State { return &state{} },
		BuildPayload: func(_ context.Context, st cc.State, _ *diag.Diagnostics) api_client.AkamaiExternalServiceConfig {
			s := st.(*state)
			return api_client.AkamaiExternalServiceConfig{
				TemplateName: s.TemplateName.ValueString(),
				IsEnabled:    s.IsEnabled.ValueBool(),
				IsDefault:    s.IsDefault.ValueBool(),
				Config: api_client.AkamaiConfig{
					AccessToken:  s.AccessToken.ValueString(),
					ClientToken:  s.ClientToken.ValueString(),
					ClientSecret: s.ClientSecret.ValueString(),
					Host:         s.Host.ValueString(),
				},
			}
		},
		Extract: func(o *api_client.AkamaiExternalServiceConfig, st cc.State) cc.APIObject {
			if o.Config.Host != "" {
				st.(*state).Host = types.StringValue(o.Config.Host)
			}
			return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault}
		},
		Create: (*api_client.APIClient).CreateAkamaiConfig,
		Get:    (*api_client.APIClient).GetAkamaiConfig,
		Update: (*api_client.APIClient).UpdateAkamaiConfig,
		Delete: (*api_client.APIClient).DeleteAkamaiConfig,
	})
}
