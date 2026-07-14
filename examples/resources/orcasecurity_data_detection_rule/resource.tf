# rule binding a DSPM policy to tagged assets (rules are created disabled by default).
# Tag selectors match asset tags by key and value; use keys = ["*"] for any key.
resource "orcasecurity_data_detection_rule" "tagged_assets" {
  name     = "Scan tagged data stores"
  policies = [orcasecurity_dspm_policy.all_data.id]
  tags = [
    { keys = ["env"], values = ["production"] }
  ]
}

# enabled rule scoped to specific cloud accounts
resource "orcasecurity_data_detection_rule" "prod_accounts" {
  name                    = "Scan production accounts"
  enabled                 = true
  action                  = "scan"
  policies                = [orcasecurity_dspm_policy.pii_eu.id]
  selector_cloud_accounts = ["12345678-aaaa-bbbb-cccc-000000000001"]
}


# Look up an existing built-in PII identifier.
data "orcasecurity_sensitive_data_identifiers" "pii" {
  category = "PII"
}
locals {
  email_identifier_id = one([
    for identifier in data.orcasecurity_sensitive_data_identifiers.pii.identifiers :
    identifier.id if identifier.title == "Email Address"
  ])
}
# Create a custom sensitive data identifier.
resource "orcasecurity_sensitive_data_identifier" "customer_id" {
  title        = "Terraform Customer ID"
  details      = "Detects customer IDs in the CUST-12345678 format."
  category     = "PII"
  sub_category = "Other"
  properties = {
    conditions = [
      {
        value = "\\b(?P<secret>CUST-[0-9]{8})\\b"
      }
    ]
    detection_types = ["text", "db"]
    sensitivity     = "high"
    significance    = "Major"
    keywords        = ["customer", "customer_id"]
    text_threshold  = 1
    db_threshold    = 1
  }
}
# Create a DSPM policy using built-in and custom identifiers.
resource "orcasecurity_dspm_policy" "customer_pii" {
  name        = "Terraform Customer PII"
  description = "Detect customer IDs and email addresses."
  tags        = ["terraform", "pii"]
  document = {
    detectors = [
      local.email_identifier_id,
      orcasecurity_sensitive_data_identifier.customer_id.id
    ]
    categories = ["PII"]
    regions    = ["Europe", "North America"]
  }
}
# Scan production assets selected by tags.
resource "orcasecurity_data_detection_rule" "production_assets" {
  name     = "Terraform scan production assets"
  enabled  = true
  action   = "scan"
  policies = [orcasecurity_dspm_policy.customer_pii.id]
  tags = [
    {
      keys   = ["environment"]
      values = ["production"]
    },
    {
      keys   = ["data-classification"]
      values = ["sensitive", "restricted"]
    }
  ]
}
# Scan assets in specific cloud accounts.
resource "orcasecurity_data_detection_rule" "cloud_accounts" {
  name     = "Terraform scan cloud accounts"
  enabled  = true
  policies = [orcasecurity_dspm_policy.customer_pii.id]
  selector_cloud_accounts = [
    "12345678-aaaa-bbbb-cccc-000000000001",
    "12345678-aaaa-bbbb-cccc-000000000002"
  ]
}
# Exclude development assets from DSPM scanning.
# A rule must always attach at least one policy (matches the Orca UI), even for
# do_not_scan exclusions.
resource "orcasecurity_data_detection_rule" "exclude_development" {
  name     = "Terraform exclude development assets"
  enabled  = true
  action   = "do_not_scan"
  policies = [orcasecurity_dspm_policy.customer_pii.id]
  tags = [
    {
      keys   = ["environment"]
      values = ["development"]
    }
  ]
}