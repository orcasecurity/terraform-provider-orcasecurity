//Cloud Provider-based Business unit for AWS
resource "orcasecurity_business_unit" "business_unit_for_aws" {
  name = "AWS"
  filter_data = {
    cloud_provider = ["aws"]
  }
}