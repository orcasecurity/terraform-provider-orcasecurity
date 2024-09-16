//2-widget custom dashboard
resource "orcasecurity_custom_dashboard" "tf-custom-dash-1" {
  name               = "Orca Custom Dashboard 1"
  filter_data        = {}
  organization_level = true
  view_type          = "dashboard"
  extra_params = {
    description = "my 1st simple dashboard"
    widgets_config = [
      {
        id   = "cloud-accounts-inventory"
        size = "sm"
      },
      {
        id   = "security-score-benchmark"
        size = "md"
      }
    ]
  }
}