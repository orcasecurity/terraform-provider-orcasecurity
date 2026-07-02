package cloudflare

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
	APIToken types.String `tfsdk:"api_token"`
}

func NewCloudflareResource() resource.Resource {
	return cc.New(cc.Spec[api_client.CloudflareExternalServiceConfig]{
		TypeNameSuffix: "_integration_cloudflare",
		UIName:         "Cloudflare integration",
		Description:    "Manage a Cloudflare integration in Orca. Creates an external service config of `service_name = \"cloudflare\"` and stores the API token in Orca's secret store.",
		VariantAttributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Cloudflare API token. Stored in Orca's secret store; never returned by the API.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
		NewState: func() cc.State { return &state{} },
		BuildPayload: func(_ context.Context, st cc.State, _ *diag.Diagnostics) api_client.CloudflareExternalServiceConfig {
			s := st.(*state)
			return api_client.CloudflareExternalServiceConfig{
				TemplateName: s.TemplateName.ValueString(),
				IsEnabled:    s.IsEnabled.ValueBool(),
				IsDefault:    s.IsDefault.ValueBool(),
				Config:       api_client.CloudflareConfig{APIToken: s.APIToken.ValueString()},
			}
		},
		Extract: func(o *api_client.CloudflareExternalServiceConfig, _ cc.State, _ *diag.Diagnostics) cc.APIObject {
			return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault}
		},
		Create: (*api_client.APIClient).CreateCloudflareConfig,
		Get:    (*api_client.APIClient).GetCloudflareConfig,
		Update: (*api_client.APIClient).UpdateCloudflareConfig,
		Delete: (*api_client.APIClient).DeleteCloudflareConfig,
	})
}
