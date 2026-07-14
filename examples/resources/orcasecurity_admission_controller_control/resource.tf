data "orcasecurity_admission_controller_template" "allowed_repos" {
  name = "k8sallowedrepos"
}

resource "orcasecurity_admission_controller_control" "allowed_repos" {
  name        = "Allowed container registries"
  description = "Only allow images from the company registry"
  template_id = data.orcasecurity_admission_controller_template.allowed_repos.id

  cluster_scope = {
    kinds = [
      {
        kinds      = ["Pod"]
        api_groups = [""]
        versions   = [""]
      }
    ]
  }

  input_parameters = jsonencode({
    repos = ["registry.example.com"]
  })
}
