# Look up a connected Jira Cloud site by name and use its id as resource_id on a Jira Cloud
# template. Jira Cloud credentials are created via the OAuth flow in the Orca UI.
data "orcasecurity_integration_jira_cloud_resource" "existing" {
  name = "my-tenant"
}

resource "orcasecurity_integration_jira_cloud_template" "demo" {
  template_name = "my-jira-cloud-template"
  resource_id   = data.orcasecurity_integration_jira_cloud_resource.existing.id
  resource_url  = data.orcasecurity_integration_jira_cloud_resource.existing.url
  project_id    = "10000"
  issue_type_id = "10001"

  mapping_json = jsonencode({
    summary = ["alert_name", "alert_id"]
  })
}
