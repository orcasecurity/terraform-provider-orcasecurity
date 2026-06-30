package azure_sentinel

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
	LogType     types.String `tfsdk:"log_type"`
	PrimaryKey  types.String `tfsdk:"primary_key"`
	WorkspaceID types.String `tfsdk:"workspace_id"`
}

func NewAzureSentinelResource() resource.Resource {
	return cc.New(cc.Spec[api_client.AzureSentinelExternalServiceConfig]{
		TypeNameSuffix:        "_integration_azure_sentinel",
		UIName:                "Azure Sentinel integration",
		Description:           "Manage an Azure Sentinel integration in Orca. Creates an external service config of `service_name = \"azure_sentinel\"`. The Log Analytics workspace primary key is stored in Orca's secret store and is never returned by the API.",
		SupportsBusinessUnits: true,
		VariantAttributes: map[string]schema.Attribute{
			"log_type": schema.StringAttribute{
				Required:    true,
				Description: "Custom log type name used in the Log Analytics workspace (for example, `OrcaAlerts`).",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"workspace_id": schema.StringAttribute{
				Required:    true,
				Description: "Azure Log Analytics workspace ID that backs the Sentinel instance.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"primary_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Azure Log Analytics workspace primary key. Stored in Orca's secret store; never returned by the API.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
		NewState: func() cc.State { return &state{} },
		BuildPayload: func(ctx context.Context, st cc.State, diags *diag.Diagnostics) api_client.AzureSentinelExternalServiceConfig {
			s := st.(*state)
			return api_client.AzureSentinelExternalServiceConfig{
				TemplateName:  s.TemplateName.ValueString(),
				IsEnabled:     s.IsEnabled.ValueBool(),
				IsDefault:     s.IsDefault.ValueBool(),
				BusinessUnits: common.BusinessUnitsToAPI(ctx, s.BusinessUnits, diags),
				Config: api_client.AzureSentinelConfig{
					LogType:     s.LogType.ValueString(),
					PrimaryKey:  s.PrimaryKey.ValueString(),
					WorkspaceID: s.WorkspaceID.ValueString(),
				},
			}
		},
		Extract: func(o *api_client.AzureSentinelExternalServiceConfig, st cc.State) cc.APIObject {
			s := st.(*state)
			if o.Config.LogType != "" {
				s.LogType = types.StringValue(o.Config.LogType)
			}
			if o.Config.WorkspaceID != "" {
				s.WorkspaceID = types.StringValue(o.Config.WorkspaceID)
			}
			return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault, BusinessUnits: o.BusinessUnits}
		},
		Create: (*api_client.APIClient).CreateAzureSentinelConfig,
		Get:    (*api_client.APIClient).GetAzureSentinelConfig,
		Update: (*api_client.APIClient).UpdateAzureSentinelConfig,
		Delete: (*api_client.APIClient).DeleteAzureSentinelConfig,
	})
}
