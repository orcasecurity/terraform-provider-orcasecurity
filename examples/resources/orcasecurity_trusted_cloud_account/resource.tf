// Trusted cloud account 
resource "orcasecurity_trusted_cloud_account" "account-1" {
  account_name      = "test44912"
  description       = "test2"
  cloud_provider    = "aws"
  cloud_provider_id = "12341234123445678912"
}