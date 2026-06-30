package torq

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/webhook_variant_common"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Torq is a webhook variant with config.type=torq. Plumbing lives in webhook_variant_common.

var torqVariant = webhook_variant_common.Variant{
	TypeNameSuffix:    "_integration_torq",
	UIName:            "Torq integration",
	WebhookConfigType: api_client.TorqWebhookType,
	CreateFn: func(c *api_client.APIClient, p api_client.WebhookExternalServiceConfig) (*api_client.WebhookExternalServiceConfig, error) {
		return c.CreateTorqConfig(p)
	},
	GetFn: func(c *api_client.APIClient, name string) (*api_client.WebhookExternalServiceConfig, error) {
		return c.GetTorqConfig(name)
	},
	UpdateFn: func(c *api_client.APIClient, name string, p api_client.WebhookExternalServiceConfig) (*api_client.WebhookExternalServiceConfig, error) {
		return c.UpdateTorqConfig(name, p)
	},
	DeleteFn: func(c *api_client.APIClient, name string) error { return c.DeleteTorqConfig(name) },
}

type torqResource struct {
	webhook_variant_common.Resource
}

func NewTorqResource() resource.Resource {
	return &torqResource{
		Resource: webhook_variant_common.Resource{Variant: torqVariant},
	}
}

func (r *torqResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = webhook_variant_common.Schema(
		"Manage a Torq integration in Orca. Orca stores Torq as a webhook variant — under the hood this resource creates an external service config of `service_name = \"webhook\"` with `type = \"torq\"`.",
		"Torq trigger webhook URL Orca posts events to.",
		"Torq API key sent with each request. Treated as sensitive.",
	)
}

var (
	_ resource.Resource                = &torqResource{}
	_ resource.ResourceWithConfigure   = &torqResource{}
	_ resource.ResourceWithImportState = &torqResource{}
)
