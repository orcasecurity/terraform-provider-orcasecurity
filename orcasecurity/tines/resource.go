package tines

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/webhook_variant_common"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Tines is a webhook variant with config.type=tines. Plumbing lives in webhook_variant_common
// via the shared config_integration_common.Spec[P] loop.

func NewTinesResource() resource.Resource {
	return webhook_variant_common.NewResource(webhook_variant_common.Options{
		TypeNameSuffix:    "_integration_tines_template",
		UIName:            "Tines integration",
		Description:       "Manage a Tines integration in Orca. Orca stores Tines as a webhook variant — under the hood this resource creates an external service config of `service_name = \"webhook\"` with `type = \"tines\"`.",
		URLDescription:    "Tines webhook URL Orca posts events to.",
		APIKeyDescription: "Tines API key sent with each request. Treated as sensitive.",
		Create:            (*api_client.APIClient).CreateTinesConfig,
		Get:               (*api_client.APIClient).GetTinesConfig,
		Update:            (*api_client.APIClient).UpdateTinesConfig,
		Delete:            (*api_client.APIClient).DeleteTinesConfig,
	})
}
