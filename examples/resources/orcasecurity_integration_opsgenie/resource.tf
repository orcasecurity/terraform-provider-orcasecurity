# Opsgenie integration
resource "orcasecurity_integration_opsgenie" "example" {
  template_name = "test_OPSGENIE"
  opsgenie_key  = var.opsgenie_key
  is_enabled    = true
  is_default    = false

  # Optional: restrict the integration to specific Orca business units.
  business_units = [
    "d354ca29-86b9-46dd-acbc-472cd5eea046",
  ]
}
