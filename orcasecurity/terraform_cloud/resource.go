package terraform_cloud

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
	APIURL   types.String `tfsdk:"api_url"`
}

func NewTerraformCloudResource() resource.Resource {
	return cc.New(cc.Spec[api_client.TerraformCloudExternalServiceConfig]{
		TypeNameSuffix: "_integration_terraform_cloud",
		UIName:         "Terraform Cloud integration",
		Description:    "Manage a Terraform Cloud (HCP Terraform) integration in Orca. Creates an external service config of `service_name = \"terraform_cloud\"`. The API token is stored in Orca's secret store and is never returned by the API.",
		VariantAttributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Terraform Cloud service-account API token. Stored in Orca's secret store; never returned by the API.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"api_url": schema.StringAttribute{
				Required:    true,
				Description: "Terraform Cloud API URL. Use `https://app.terraform.io` for HCP Terraform or your Terraform Enterprise hostname.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
		NewState: func() cc.State { return &state{} },
		BuildPayload: func(_ context.Context, st cc.State, _ *diag.Diagnostics) api_client.TerraformCloudExternalServiceConfig {
			s := st.(*state)
			return api_client.TerraformCloudExternalServiceConfig{
				TemplateName: s.TemplateName.ValueString(),
				IsEnabled:    s.IsEnabled.ValueBool(),
				IsDefault:    s.IsDefault.ValueBool(),
				Config:       api_client.TerraformCloudConfig{APIToken: s.APIToken.ValueString(), APIURL: s.APIURL.ValueString()},
			}
		},
		Extract: func(o *api_client.TerraformCloudExternalServiceConfig, st cc.State, _ *diag.Diagnostics) cc.APIObject {
			if o.Config.APIURL != "" {
				st.(*state).APIURL = types.StringValue(o.Config.APIURL)
			}
			return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault}
		},
		Create: (*api_client.APIClient).CreateTerraformCloudConfig,
		Get:    (*api_client.APIClient).GetTerraformCloudConfig,
		Update: (*api_client.APIClient).UpdateTerraformCloudConfig,
		Delete: (*api_client.APIClient).DeleteTerraformCloudConfig,
	})
}
