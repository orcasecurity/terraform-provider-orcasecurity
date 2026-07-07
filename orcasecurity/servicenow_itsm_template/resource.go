package servicenow_itsm_template

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/servicenow_template_common"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

const itsmDescription = "Manage a ServiceNow ITSM template in Orca. Creates an external service config of `service_name = \"sn_incidents\"` with `config.type = \"ITSM\"` and links it to the credentials side of the integration (an `orcasecurity_integration_servicenow` resource). Templates carry the per-ticket settings — field mapping, resolution codes, reopen behaviour. Inspect the available ITSM fields with the `orcasecurity_integration_servicenow_schema` data source (`type = \"itsm\"`)."

func NewServiceNowITSMTemplateResource() resource.Resource {
	return servicenow_template_common.NewResource(servicenow_template_common.Options{
		TypeNameSuffix: "_integration_servicenow_itsm_template",
		UIName:         "ServiceNow ITSM template",
		Description:    itsmDescription,
		ConfigType:     api_client.ServiceNowITSMTemplateConfigType,
		Create:         (*api_client.APIClient).CreateServiceNowITSMTemplate,
		Get:            (*api_client.APIClient).GetServiceNowITSMTemplate,
		Update:         (*api_client.APIClient).UpdateServiceNowITSMTemplate,
		Delete:         (*api_client.APIClient).DeleteServiceNowITSMTemplate,
	})
}
