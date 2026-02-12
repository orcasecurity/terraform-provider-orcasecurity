resource "orcasecurity_custom_dashboard" "my_dashboard" {
  name               = "Dashboard with Custom Widget"
  organization_level = true
  filter_data        = {}
  view_type          = "dashboard"
  extra_params = {
    description = ""
    widgets_config = [
      { id = orcasecurity_custom_widget.my_widget.id, size = "sm" },
      { id = "cloud-accounts-inventory", size = "sm" }
    ]
  }
}
