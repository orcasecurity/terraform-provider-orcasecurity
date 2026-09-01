# Look up an existing user-marked crown jewel (e.g. before import).
data "orcasecurity_crown_jewel" "example" {
  group_unique_id = "vm_123456789012_i-0123456789abcdef0"
}

output "reason" {
  value = data.orcasecurity_crown_jewel.example.description
}
