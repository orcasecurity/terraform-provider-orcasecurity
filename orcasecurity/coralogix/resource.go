package coralogix

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/webhook_variant_common"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Coralogix is just a webhook with config.type pinned. All CRUD plumbing lives in
// webhook_variant_common.

var coralogixVariant = webhook_variant_common.Variant{
	TypeNameSuffix:    "_integration_coralogix",
	UIName:            "Coralogix integration",
	WebhookConfigType: api_client.CoralogixWebhookType,
	CreateFn: func(c *api_client.APIClient, p api_client.WebhookExternalServiceConfig) (*api_client.WebhookExternalServiceConfig, error) {
		return c.CreateCoralogixConfig(p)
	},
	GetFn: func(c *api_client.APIClient, name string) (*api_client.WebhookExternalServiceConfig, error) {
		return c.GetCoralogixConfig(name)
	},
	UpdateFn: func(c *api_client.APIClient, name string, p api_client.WebhookExternalServiceConfig) (*api_client.WebhookExternalServiceConfig, error) {
		return c.UpdateCoralogixConfig(name, p)
	},
	DeleteFn: func(c *api_client.APIClient, name string) error { return c.DeleteCoralogixConfig(name) },
}

type coralogixResource struct {
	webhook_variant_common.Resource
}

func NewCoralogixResource() resource.Resource {
	return &coralogixResource{
		Resource: webhook_variant_common.Resource{Variant: coralogixVariant},
	}
}

func (r *coralogixResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = webhook_variant_common.Schema(
		"Manage a Coralogix integration in Orca. Orca stores Coralogix as a webhook variant — under the hood this resource creates an external service config of `service_name = \"webhook\"` with `type = \"coralogix\"`.",
		"Coralogix ingest URL Orca posts events to (for example, `https://coralogix.us`).",
		"Coralogix API key sent with each request. Treated as sensitive.",
	)
}

var (
	_ resource.Resource                = &coralogixResource{}
	_ resource.ResourceWithConfigure   = &coralogixResource{}
	_ resource.ResourceWithImportState = &coralogixResource{}
)
