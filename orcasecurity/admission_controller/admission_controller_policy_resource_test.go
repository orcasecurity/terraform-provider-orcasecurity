package admission_controller_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccPolicyControlConfig = `
data "orcasecurity_admission_controller_template" "repos" {
  name = "k8sallowedrepos"
}

resource "orcasecurity_admission_controller_control" "for_policy" {
  name        = "tf-acc-policy-control"
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
`

func TestAccAdmissionControllerPolicyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with defaults (is_active, enforcement_action omitted)
			{
				Config: orcasecurity.TestProviderConfig + testAccPolicyControlConfig + `
resource "orcasecurity_admission_controller_policy" "test" {
  name     = "tf-acc-policy-1"
  controls = [orcasecurity_admission_controller_control.for_policy.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("orcasecurity_admission_controller_policy.test", "id"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy.test", "is_active", "true"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy.test", "enforcement_action", "monitor"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy.test", "controls.#", "1"),
				),
			},
			// ImportState
			{
				ResourceName:      "orcasecurity_admission_controller_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update: flip enforcement + deactivate
			{
				Config: orcasecurity.TestProviderConfig + testAccPolicyControlConfig + `
resource "orcasecurity_admission_controller_policy" "test" {
  name               = "tf-acc-policy-1-renamed"
  description        = "updated"
  is_active          = false
  enforcement_action = "block"
  controls           = [orcasecurity_admission_controller_control.for_policy.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy.test", "name", "tf-acc-policy-1-renamed"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy.test", "is_active", "false"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy.test", "enforcement_action", "block"),
				),
			},
		},
	})
}
