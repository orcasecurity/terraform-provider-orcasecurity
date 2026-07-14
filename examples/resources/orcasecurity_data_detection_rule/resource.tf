# rule binding a DSPM policy to tagged assets (rules are created disabled by default)
resource "orcasecurity_data_detection_rule" "tagged_assets" {
  name     = "Scan tagged data stores"
  policies = [orcasecurity_dspm_policy.all_data.id]
  tags     = ["env:production"]
}

# enabled rule scoped to specific cloud accounts
resource "orcasecurity_data_detection_rule" "prod_accounts" {
  name                    = "Scan production accounts"
  enabled                 = true
  action                  = "scan"
  policies                = [orcasecurity_dspm_policy.pii_eu.id]
  selector_cloud_accounts = ["12345678-aaaa-bbbb-cccc-000000000001"]
}
