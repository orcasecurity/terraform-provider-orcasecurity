package admission_controller_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAdmissionControllerControlResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_admission_controller_template" "repos" {
  name = "k8sallowedrepos"
}

resource "orcasecurity_admission_controller_control" "test" {
  name        = "tf-acc-control-1"
  description = "acceptance test control"
  template_id = data.orcasecurity_admission_controller_template.repos.id
  cluster_scope = {
    kinds = [
      {
        kinds      = ["Pod"]
        api_groups = [""]
        versions   = [""]
      }
    ]
  }
  input_parameters = jsonencode({ repos = ["docker.io/library"] })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("orcasecurity_admission_controller_control.test", "id"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_control.test", "name", "tf-acc-control-1"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_control.test", "template_name", "k8sallowedrepos"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_control.test", "cluster_scope.kinds.0.kinds.0", "Pod"),
				),
			},
			// ImportState
			{
				ResourceName:      "orcasecurity_admission_controller_control.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update: rename + change input_parameters
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_admission_controller_template" "repos" {
  name = "k8sallowedrepos"
}

resource "orcasecurity_admission_controller_control" "test" {
  name        = "tf-acc-control-1-renamed"
  description = "acceptance test control updated"
  template_id = data.orcasecurity_admission_controller_template.repos.id
  cluster_scope = {
    kinds = [
      {
        kinds      = ["Pod"]
        api_groups = [""]
        versions   = [""]
      }
    ]
  }
  input_parameters = jsonencode({ repos = ["docker.io/library", "gcr.io/project"] })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_control.test", "name", "tf-acc-control-1-renamed"),
				),
			},
			// Destroy happens automatically at the end of the test.
		},
	})
}
