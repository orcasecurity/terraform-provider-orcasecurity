package pagerduty

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
	IntegrationKey types.String `tfsdk:"integration_key"`
}

// buildPayload converts the planned state into the PagerDuty API payload.
func buildPayload(_ context.Context, st cc.State, _ *diag.Diagnostics) api_client.PagerDutyExternalServiceConfig {
	s := st.(*state)
	return api_client.PagerDutyExternalServiceConfig{
		TemplateName: s.TemplateName.ValueString(),
		IsEnabled:    s.IsEnabled.ValueBool(),
		IsDefault:    s.IsDefault.ValueBool(),
		Config:       api_client.PagerDutyConfig{IntegrationKey: s.IntegrationKey.ValueString()},
	}
}

// extract maps the API envelope back onto state; the key is never returned so state is untouched.
func extract(o *api_client.PagerDutyExternalServiceConfig, _ cc.State, _ *diag.Diagnostics) cc.APIObject {
	return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault}
}

func NewPagerDutyResource() resource.Resource {
	return cc.New(cc.Spec[api_client.PagerDutyExternalServiceConfig]{
		TypeNameSuffix: "_integration_pagerduty",
		UIName:         "PagerDuty integration",
		Description:    "Manage a PagerDuty integration in Orca. Creates an external service config of `service_name = \"pagerduty\"` and stores the integration key in Orca's secret store.",
		VariantAttributes: map[string]schema.Attribute{
			"integration_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "PagerDuty Events API V2 integration key. Stored in Orca's secret store; never returned by the API.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
		NewState:     func() cc.State { return &state{} },
		BuildPayload: buildPayload,
		Extract:      extract,
		Create:       (*api_client.APIClient).CreatePagerDutyConfig,
		Get:          (*api_client.APIClient).GetPagerDutyConfig,
		Update:       (*api_client.APIClient).UpdatePagerDutyConfig,
		Delete:       (*api_client.APIClient).DeletePagerDutyConfig,
	})
}
