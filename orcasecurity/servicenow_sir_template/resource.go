package servicenow_sir_template

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/servicenow_template_common"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

const sirDescription = "Manage a ServiceNow SIR (Security Incident Response) template in Orca. Same shape as the ITSM template — the only difference at the API level is `config.type = \"SIR\"`. Inspect the available SIR fields with the `orcasecurity_integration_servicenow_sir_schema` data source."

func NewServiceNowSIRTemplateResource() resource.Resource {
	return servicenow_template_common.NewResource(servicenow_template_common.Options{
		TypeNameSuffix: "_integration_servicenow_sir_template",
		UIName:         "ServiceNow SIR template",
		Description:    sirDescription,
		ConfigType:     api_client.ServiceNowSIRTemplateConfigType,
		Create:         (*api_client.APIClient).CreateServiceNowSIRTemplate,
		Get:            (*api_client.APIClient).GetServiceNowSIRTemplate,
		Update:         (*api_client.APIClient).UpdateServiceNowSIRTemplate,
		Delete:         (*api_client.APIClient).DeleteServiceNowSIRTemplate,
	})
}
