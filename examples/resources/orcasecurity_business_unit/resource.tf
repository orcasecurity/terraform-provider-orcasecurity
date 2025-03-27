// Cloud Provider-based Business Unit for AWS and all Shift Left projects
resource "orcasecurity_business_unit" "example" {
  name = "AWS and Shift Left"

  filter_data = {
    cloud_providers = ["aws", "shiftleft"]
  }
}

// Cloud Tag-based Business Unit for All Resources with the Env:Prod tag or Sensitive:True tag.
resource "orcasecurity_business_unit" "BU-Tags" {
  name = "BU-Tags"

  filter_data = {
    cloud_tags = [
      "Orca|True"
    ]
  }
}

// Cloud Provider-based Business Unit for AWS and all Shift Left projects
resource "orcasecurity_business_unit" "BU-OrcaTags" {
  name = "BU-OrcaTags"

  filter_data = {
    custom_tags = [
      "critical|true"
    ]
  }
}