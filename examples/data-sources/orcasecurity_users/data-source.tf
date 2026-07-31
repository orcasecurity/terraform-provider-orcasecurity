data "orcasecurity_users" "all" {}

output "jane_user_id" {
  value = one([for u in data.orcasecurity_users.all.users : u.user_id if u.email == "jane@example.com"])
}
