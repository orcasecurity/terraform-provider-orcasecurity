data "orcasecurity_automation_v2_priorities" "all" {}

output "automation_order" {
  value = [for a in data.orcasecurity_automation_v2_priorities.all.automations : "${a.priority}: ${a.name}"]
}
