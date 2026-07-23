variable "azure_devops_token" {
  type      = string
  sensitive = true
}

# Token scoped to a single Azure DevOps organization
resource "orcasecurity_shift_left_azure_devops_installation" "single_org" {
  name         = "Azure DevOps"
  access_token = var.azure_devops_token
  account_name = "my-organization"
}

# All-organizations token on a self-hosted server
resource "orcasecurity_shift_left_azure_devops_installation" "server" {
  name         = "Azure DevOps Server"
  server_url   = "https://azuredevops.example.com"
  access_token = var.azure_devops_token
}
