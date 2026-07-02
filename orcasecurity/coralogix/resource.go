package coralogix

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/webhook_variant_common"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Coralogix is a webhook variant with config.type=coralogix. Plumbing lives in
// webhook_variant_common via the shared config_integration_common.Spec[P] loop.

func NewCoralogixResource() resource.Resource {
	return webhook_variant_common.NewResource(webhook_variant_common.Options{
		TypeNameSuffix:    "_integration_coralogix",
		UIName:            "Coralogix integration",
		Description:       "Manage a Coralogix integration in Orca. Orca stores Coralogix as a webhook variant — under the hood this resource creates an external service config of `service_name = \"webhook\"` with `type = \"coralogix\"`.",
		URLDescription:    "Coralogix ingest URL Orca posts events to (for example, `https://coralogix.us`).",
		APIKeyDescription: "Coralogix API key sent with each request. Treated as sensitive.",
		Create:            (*api_client.APIClient).CreateCoralogixConfig,
		Get:               (*api_client.APIClient).GetCoralogixConfig,
		Update:            (*api_client.APIClient).UpdateCoralogixConfig,
		Delete:            (*api_client.APIClient).DeleteCoralogixConfig,
	})
}
