package splunk

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type state struct {
	cc.CommonFields
	URL                 types.String `tfsdk:"url"`
	Token               types.String `tfsdk:"token"`
	AllowSelfSignedCert types.Bool   `tfsdk:"allow_self_signed_cert"`
}

func NewSplunkResource() resource.Resource {
	return cc.New(cc.Spec[api_client.SplunkExternalServiceConfig]{
		TypeNameSuffix: "_integration_splunk",
		UIName:         "Splunk integration",
		Description:    "Manage a Splunk HEC integration in Orca. Creates an external service config of `service_name = \"splunk\"`. The HEC token is stored in Orca's secret store and is never returned by the API.",
		VariantAttributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Required:    true,
				Description: "Splunk HEC endpoint URL (for example, `https://prd-p-xxxxx.splunkcloud.com:8088/services/collector/event`).",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Splunk HEC token. Stored in Orca's secret store; never returned by the API.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"allow_self_signed_cert": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether Orca should accept self-signed TLS certificates when calling the Splunk endpoint. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
		},
		NewState: func() cc.State { return &state{} },
		BuildPayload: func(_ context.Context, st cc.State, _ *diag.Diagnostics) api_client.SplunkExternalServiceConfig {
			s := st.(*state)
			return api_client.SplunkExternalServiceConfig{
				TemplateName: s.TemplateName.ValueString(),
				IsEnabled:    s.IsEnabled.ValueBool(),
				IsDefault:    s.IsDefault.ValueBool(),
				Config: api_client.SplunkConfig{
					URL:                 s.URL.ValueString(),
					Token:               s.Token.ValueString(),
					AllowSelfSignedCert: s.AllowSelfSignedCert.ValueBool(),
				},
			}
		},
		Extract: func(o *api_client.SplunkExternalServiceConfig, st cc.State) cc.APIObject {
			s := st.(*state)
			if o.Config.URL != "" {
				s.URL = types.StringValue(o.Config.URL)
			}
			s.AllowSelfSignedCert = types.BoolValue(o.Config.AllowSelfSignedCert)
			return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault}
		},
		Create: (*api_client.APIClient).CreateSplunkConfig,
		Get:    (*api_client.APIClient).GetSplunkConfig,
		Update: (*api_client.APIClient).UpdateSplunkConfig,
		Delete: (*api_client.APIClient).DeleteSplunkConfig,
	})
}
