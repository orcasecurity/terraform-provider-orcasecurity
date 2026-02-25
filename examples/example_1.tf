terraform {
  required_providers {
    orcasecurity = {
      source = "orcasecurity/orcasecurity"
    }
  }
}

provider "orcasecurity" {
  api_endpoint = "https://api.orcasecurity.io"
  api_token    = "myorcatoken"
}
