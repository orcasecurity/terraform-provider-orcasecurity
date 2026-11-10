variable "bitbucket_token" {
  type      = string
  sensitive = true
}

# Bitbucket cloud workspace token
resource "orcasecurity_shift_left_bitbucket_installation" "cloud" {
  name              = "Bitbucket Cloud"
  access_token      = var.bitbucket_token
  access_token_type = "TOKEN"
  account_id        = "my-workspace"
}

# Bitbucket server personal access token
resource "orcasecurity_shift_left_bitbucket_installation" "server" {
  name              = "Bitbucket Server"
  server_url        = "https://bitbucket.example.com"
  access_token      = var.bitbucket_token
  access_token_type = "PAT"
  username          = "svc-orca"
}
