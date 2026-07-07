package torq

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/webhook_variant_common"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Torq is a webhook variant with config.type=torq. Plumbing lives in webhook_variant_common
// via the shared config_integration_common.Spec[P] loop.

func NewTorqResource() resource.Resource {
	return webhook_variant_common.NewResource(webhook_variant_common.Options{
		TypeNameSuffix:    "_integration_torq_template",
		UIName:            "Torq integration",
		Description:       "Manage a Torq integration in Orca. Orca stores Torq as a webhook variant — under the hood this resource creates an external service config of `service_name = \"webhook\"` with `type = \"torq\"`.",
		URLDescription:    "Torq trigger webhook URL Orca posts events to.",
		APIKeyDescription: "Torq API key sent with each request. Treated as sensitive.",
		Create:            (*api_client.APIClient).CreateTorqConfig,
		Get:               (*api_client.APIClient).GetTorqConfig,
		Update:            (*api_client.APIClient).UpdateTorqConfig,
		Delete:            (*api_client.APIClient).DeleteTorqConfig,
	})
}
