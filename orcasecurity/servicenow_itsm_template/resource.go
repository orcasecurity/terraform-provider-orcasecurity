package servicenow_itsm_template

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/servicenow_template_common"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// All the heavy lifting (schema, model, CRUD plumbing) lives in
// servicenow_template_common; this file only declares the ITSM-specific variant constants
// and wires them up through the embeddable base resource.

var itsmVariant = servicenow_template_common.Variant{
	TypeNameSuffix: "_integration_servicenow_itsm_template",
	UIName:         "ServiceNow ITSM template",
	Create: func(c *api_client.APIClient, p api_client.ServiceNowITSMTemplate) (*api_client.ServiceNowITSMTemplate, error) {
		return c.CreateServiceNowITSMTemplate(p)
	},
	Get: func(c *api_client.APIClient, name string) (*api_client.ServiceNowITSMTemplate, error) {
		return c.GetServiceNowITSMTemplate(name)
	},
	Update: func(c *api_client.APIClient, name string, p api_client.ServiceNowITSMTemplate) (*api_client.ServiceNowITSMTemplate, error) {
		return c.UpdateServiceNowITSMTemplate(name, p)
	},
	Delete: func(c *api_client.APIClient, name string) error { return c.DeleteServiceNowITSMTemplate(name) },
}

const itsmDescription = "Manage a ServiceNow ITSM template in Orca. Creates an external service config of `service_name = \"sn_incidents\"` with `config.type = \"ITSM\"` and links it to the credentials side of the integration (an `orcasecurity_integration_servicenow` resource). Templates carry the per-ticket settings — field mapping, resolution codes, reopen behaviour."

type itsmResource struct {
	servicenow_template_common.Resource
}

func NewServiceNowITSMTemplateResource() resource.Resource {
	return &itsmResource{
		Resource: servicenow_template_common.Resource{
			Variant:    itsmVariant,
			ConfigType: "ITSM",
		},
	}
}

func (r *itsmResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = servicenow_template_common.Schema(itsmDescription)
}

var (
	_ resource.Resource                = &itsmResource{}
	_ resource.ResourceWithConfigure   = &itsmResource{}
	_ resource.ResourceWithImportState = &itsmResource{}
)
