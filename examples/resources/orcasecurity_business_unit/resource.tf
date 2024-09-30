//Cloud Provider-based Business unit for AWS
resource "orcasecurity_business_unit" "business_unit_for_aws" {
  name = "AWS"

  shiftleft_filter_data = {
    shiftleft_project_id = [
      "c7257bec-9718-47c2-ade7-e08e7caa36e3"
    ]
  }

  filter_data = {
    cloud_provider = ["aws"]
  }
}