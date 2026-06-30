package servicenow_sir_template

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/servicenow_template_common"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// SIR-specific variant constants. Heavy lifting lives in servicenow_template_common.

var sirVariant = servicenow_template_common.Variant{
	TypeNameSuffix: "_integration_servicenow_sir_template",
	UIName:         "ServiceNow SIR template",
	Create: func(c *api_client.APIClient, p api_client.ServiceNowITSMTemplate) (*api_client.ServiceNowITSMTemplate, error) {
		return c.CreateServiceNowSIRTemplate(p)
	},
	Get: func(c *api_client.APIClient, name string) (*api_client.ServiceNowITSMTemplate, error) {
		return c.GetServiceNowSIRTemplate(name)
	},
	Update: func(c *api_client.APIClient, name string, p api_client.ServiceNowITSMTemplate) (*api_client.ServiceNowITSMTemplate, error) {
		return c.UpdateServiceNowSIRTemplate(name, p)
	},
	Delete: func(c *api_client.APIClient, name string) error { return c.DeleteServiceNowSIRTemplate(name) },
}

const sirDescription = "Manage a ServiceNow SIR (Security Incident Response) template in Orca. Same shape as the ITSM template — the only difference at the API level is `config.type = \"SIR\"`. Inspect the available SIR fields with the `orcasecurity_integration_servicenow_sir_schema` data source."

type sirResource struct {
	servicenow_template_common.Resource
}

func NewServiceNowSIRTemplateResource() resource.Resource {
	return &sirResource{
		Resource: servicenow_template_common.Resource{
			Variant:    sirVariant,
			ConfigType: api_client.ServiceNowSIRTemplateConfigType,
		},
	}
}

func (r *sirResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = servicenow_template_common.Schema(sirDescription)
}

var (
	_ resource.Resource                = &sirResource{}
	_ resource.ResourceWithConfigure   = &sirResource{}
	_ resource.ResourceWithImportState = &sirResource{}
)
