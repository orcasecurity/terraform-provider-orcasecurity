# Azure Sentinel integration
resource "orcasecurity_integration_azure_sentinel" "example" {
  template_name = "Azure Sentinel name"
  log_type      = "OrcaAlerts"
  workspace_id  = "workspace_id"
  primary_key   = var.azure_sentinel_primary_key
  is_enabled    = true
  is_default    = false

  business_units = [
    "d7a3b159-3063-433b-b954-0586ff3e8438",
    "5308642b-f207-4610-a13b-f39c4db4a7a3",
  ]
}
