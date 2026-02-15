# Mix built-in widgets (string IDs) and custom widgets (resource IDs).
# Built-in: use string ID from Orca docs. Custom: use orcasecurity_custom_widget.<name>.id
resource "orcasecurity_custom_dashboard" "my_dashboard" {
  name               = "Dashboard with Built-in and Custom Widgets"
  organization_level = true
  filter_data        = {}
  view_type          = "dashboard"
  extra_params = {
    description = "Built-in and custom widgets together"
    version     = 2
    widgets_config = [
      # Built-in widgets — string IDs from Orca docs
      { id = "cloud-accounts-inventory", size = "sm" },
      { id = "security-score-benchmark", size = "md" },
      { id = "alerts-by-severity", size = "sm" },
      # Custom widget — reference the resource
      { id = orcasecurity_custom_widget.my_widget.id, size = "sm" }
    ]
  }
}
