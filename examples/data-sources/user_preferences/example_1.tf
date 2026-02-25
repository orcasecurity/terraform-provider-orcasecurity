data "orcasecurity_user_preferences" "widgets" {}

# List of widget IDs (strings)
output "custom_widget_ids" {
  value = data.orcasecurity_user_preferences.widgets.custom_widget_ids
}

# List of objects with id and name
output "widgets" {
  value = data.orcasecurity_user_preferences.widgets.custom_widgets
}
