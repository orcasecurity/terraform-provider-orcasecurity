//Group
resource "orcasecurity_group" "tf-group-1" {
    name = "Orca Terraform Group 1"
    
    sso_group = true
    description = "string"
    users = [
        "{place-user-id-here}"
    ]
}