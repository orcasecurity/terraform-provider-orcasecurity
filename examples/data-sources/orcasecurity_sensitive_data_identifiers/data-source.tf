# look up built-in and custom identifiers in the PII category
data "orcasecurity_sensitive_data_identifiers" "pii" {
  category = "PII"
}

# reference a built-in identifier id by title
locals {
  email_identifier_id = one([
    for identifier in data.orcasecurity_sensitive_data_identifiers.pii.identifiers :
    identifier.id if identifier.title == "Email Address"
  ])
}
