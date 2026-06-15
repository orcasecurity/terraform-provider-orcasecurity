//Saved discovery view
resource "orcasecurity_discovery_view" "tf-disco-view-1" {
  name = "orca-disco-view-1"

  organization_level = true
  view_type          = "discovery"
  extra_params       = {}
  filter_data = {
    query = jsonencode({
      "models" : [
        "AwsS3Bucket"
      ],
      "type" : "object_set"
    })
  }
}

//Saved discovery view with custom display columns
resource "orcasecurity_discovery_view" "tf-disco-view-with-columns" {
  name = "orca-disco-view-with-columns"

  organization_level = true
  view_type          = "discovery"
  extra_params       = {}

  # Control the columns (and their order) shown in the view.
  # Keys are Sonar field names or special aggregate columns ($-prefixed).
  # Valid keys depend on the models in filter_data.query; the exact set for a
  # view can be read from GET /api/user_preferences/{id} -> extra_params.columns2.keys
  columns = [
    "$overview",
    "CloudAccount",
    "OrcaScore",
    "$alertsStats",
    "$attackPaths",
    "Exposure",
    "SensitiveData",
    "Tags",
    "AssetUniqueId",
    "ConsoleUrlLink",
    "AccessEndpoints",
  ]

  # Sort by OrcaScore descending ("-" prefix), and group results by AlertType.
  sort     = "-OrcaScore"
  group_by = ["AlertType"]

  filter_data = {
    query = jsonencode({
      "models" : [
        "AwsEc2Instance",
        "AzureComputeVm",
        "GcpVmInstance",
      ],
      "type" : "object_set"
    })
  }
}
