# Tines integration
resource "orcasecurity_integration_tines_template" "example" {
  template_name = "tines_name"
  webhook_url   = "https://your-tenant.tines.com/webhook/<path>/<secret>"
  api_key       = var.tines_api_key
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
