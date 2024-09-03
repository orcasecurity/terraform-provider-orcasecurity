//Saved discovery view
resource "orcasecurity_discovery_view" "tf-disco-view-1" {
    name = "orca-disco-view-1"

    organization_level = true
    view_type = "discovery"
    extra_params = {}
    filter_data = {
        query = jsonencode({
            "models": [
                "AwsS3Bucket"
            ],
            "type": "object_set"
        })
    }
}