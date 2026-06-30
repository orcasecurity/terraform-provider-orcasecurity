package opsgenie

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type state struct {
	cc.CommonFieldsWithBU
	OpsgenieKey types.String `tfsdk:"opsgenie_key"`
}

func NewOpsgenieResource() resource.Resource {
	return cc.New(cc.Spec[api_client.OpsgenieExternalServiceConfig]{
		TypeNameSuffix:        "_integration_opsgenie",
		UIName:                "Opsgenie integration",
		Description:           "Manage an Opsgenie integration in Orca. Creates an external service config of `service_name = \"opsgenie\"` and stores the Opsgenie API key in Orca's secret store.",
		SupportsBusinessUnits: true,
		VariantAttributes: map[string]schema.Attribute{
			"opsgenie_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Opsgenie API integration key. Stored in Orca's secret store; never returned by the API.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
		NewState: func() cc.State { return &state{} },
		BuildPayload: func(ctx context.Context, st cc.State, diags *diag.Diagnostics) api_client.OpsgenieExternalServiceConfig {
			s := st.(*state)
			return api_client.OpsgenieExternalServiceConfig{
				TemplateName:  s.TemplateName.ValueString(),
				IsEnabled:     s.IsEnabled.ValueBool(),
				IsDefault:     s.IsDefault.ValueBool(),
				Config:        api_client.OpsgenieConfig{OpsgenieKey: s.OpsgenieKey.ValueString()},
				BusinessUnits: common.BusinessUnitsToAPI(ctx, s.BusinessUnits, diags),
			}
		},
		Extract: func(o *api_client.OpsgenieExternalServiceConfig, _ cc.State) cc.APIObject {
			return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault, BusinessUnits: o.BusinessUnits}
		},
		Create: (*api_client.APIClient).CreateOpsgenieConfig,
		Get:    (*api_client.APIClient).GetOpsgenieConfig,
		Update: (*api_client.APIClient).UpdateOpsgenieConfig,
		Delete: (*api_client.APIClient).DeleteOpsgenieConfig,
	})
}
