variable "gitlab_token" {
  type      = string
  sensitive = true
}

# GitLab cloud (gitlab.com)
resource "orcasecurity_shift_left_gitlab_installation" "cloud" {
  name         = "GitLab Cloud"
  access_token = var.gitlab_token
}

# Self-managed GitLab with a read-only token
resource "orcasecurity_shift_left_gitlab_installation" "self_managed" {
  name         = "GitLab Enterprise"
  server_url   = "https://gitlab.example.com"
  access_token = var.gitlab_token
  read_only    = true
}
