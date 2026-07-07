# Torq integration
resource "orcasecurity_integration_torq_template" "example" {
  template_name = "torq_name"
  webhook_url   = "https://trigger-url.com"
  api_key       = var.torq_api_key
  is_enabled    = true
  is_default    = true

  business_units = [
    "d7a3b159-3063-433b-b954-0586ff3e8438",
    "003b5734-131f-4d3a-8bfa-abaed4d139fe",
    "beac1ac8-dc1c-40cc-853c-a368d282bdef",
  ]

  custom_headers = {
    key_header_1 = [{ custom = "value_header_1" }]
  }
}
