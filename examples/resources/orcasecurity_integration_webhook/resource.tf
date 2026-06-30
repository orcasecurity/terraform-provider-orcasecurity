# Webhook integration
resource "orcasecurity_integration_webhook" "example" {
  template_name = "webhook_name"
  is_enabled    = true
  is_default    = true

  business_units = [
    "d7a3b159-3063-433b-b954-0586ff3e8438",
    "003b5734-131f-4d3a-8bfa-abaed4d139fe",
    "beac1ac8-dc1c-40cc-853c-a368d282bdef",
  ]

  config = {
    webhook_url = "https://webhook.site/0f339e1a-0026-432a-8ad7-d3304086a2e5"
    type        = "common"
    api_key     = var.webhook_api_key
    body_fields = []

    custom_headers = {
      key_head_1 = [{ custom = "value_head_1" }]
      key_head_2 = [{ custom = "value_head_2" }]
    }
  }
}
