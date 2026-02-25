data "orcasecurity_user_preferences" "widgets" {}

output "custom_widgets" {
  value = data.orcasecurity_user_preferences.widgets.custom_widgets
}
