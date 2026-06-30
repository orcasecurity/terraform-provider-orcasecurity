package snyk

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

var snykRegions = []string{"US", "EU", "AU", "US2"}

type state struct {
	cc.CommonFields
	APIToken types.String `tfsdk:"api_token"`
	Region   types.String `tfsdk:"region"`
}

func NewSnykResource() resource.Resource {
	return cc.New(cc.Spec[api_client.SnykExternalServiceConfig]{
		TypeNameSuffix: "_integration_snyk",
		UIName:         "Snyk integration",
		Description:    "Manage a Snyk integration in Orca. Creates an external service config of `service_name = \"snyk\"`. The Snyk API token is stored in Orca's secret store and is never returned by the API.",
		VariantAttributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Snyk service account API token. Stored in Orca's secret store; never returned by the API.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"region": schema.StringAttribute{
				Required: true,
				Description: "Snyk tenant region. The Orca UI exposes four choices — pass the matching API code here:\n" +
					"  - `US`  — United States (app.snyk.io)\n" +
					"  - `US2` — United States 2 (app.us.snyk.io)\n" +
					"  - `EU`  — European Union (app.eu.snyk.io)\n" +
					"  - `AU`  — Australia (app.au.snyk.io)",
				Validators: []validator.String{stringvalidator.OneOf(snykRegions...)},
			},
		},
		NewState: func() cc.State { return &state{} },
		BuildPayload: func(_ context.Context, st cc.State, _ *diag.Diagnostics) api_client.SnykExternalServiceConfig {
			s := st.(*state)
			return api_client.SnykExternalServiceConfig{
				TemplateName: s.TemplateName.ValueString(),
				IsEnabled:    s.IsEnabled.ValueBool(),
				IsDefault:    s.IsDefault.ValueBool(),
				Config:       api_client.SnykConfig{APIToken: s.APIToken.ValueString(), Region: s.Region.ValueString()},
			}
		},
		Extract: func(o *api_client.SnykExternalServiceConfig, st cc.State) cc.APIObject {
			if o.Config.Region != "" {
				st.(*state).Region = types.StringValue(o.Config.Region)
			}
			return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault}
		},
		Create: (*api_client.APIClient).CreateSnykConfig,
		Get:    (*api_client.APIClient).GetSnykConfig,
		Update: (*api_client.APIClient).UpdateSnykConfig,
		Delete: (*api_client.APIClient).DeleteSnykConfig,
	})
}
