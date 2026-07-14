# policy covering all identifiers
resource "orcasecurity_dspm_policy" "all_data" {
  name        = "All sensitive data"
  description = "Scan for every sensitive data identifier."
  document = {
    detectors = ["*"]
  }
}

# policy covering specific built-in and custom identifiers
resource "orcasecurity_dspm_policy" "pii_eu" {
  name        = "EU PII"
  description = "PII identifiers scoped to EU data."
  tags        = ["compliance:gdpr"]
  document = {
    detectors  = ["PII-Email", orcasecurity_sensitive_data_identifier.account_number.id]
    categories = ["PII"]
    regions    = ["EU"]
  }
}
