package admission_controller_test

import (
	"regexp"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccAssignmentPolicyConfig = `
data "orcasecurity_admission_controller_template" "repos_for_assignment" {
  name = "k8sallowedrepos"
}

resource "orcasecurity_admission_controller_control" "for_assignment" {
  name        = "tf-acc-assignment-control"
  template_id = data.orcasecurity_admission_controller_template.repos_for_assignment.id
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

resource "orcasecurity_admission_controller_policy" "for_assignment" {
  name     = "tf-acc-assignment-policy"
  controls = [orcasecurity_admission_controller_control.for_assignment.id]
}
`

func TestAccAdmissionControllerPolicyAssignmentResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create (full organization scope)
			{
				Config: orcasecurity.TestProviderConfig + testAccAssignmentPolicyConfig + `
resource "orcasecurity_admission_controller_policy_assignment" "test" {
  name              = "tf-acc-assignment-1"
  full_organization = true
  policy_ids        = [orcasecurity_admission_controller_policy.for_assignment.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("orcasecurity_admission_controller_policy_assignment.test", "id"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy_assignment.test", "full_organization", "true"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy_assignment.test", "policy_ids.#", "1"),
				),
			},
			// ImportState
			{
				ResourceName:      "orcasecurity_admission_controller_policy_assignment.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update: rename + description
			{
				Config: orcasecurity.TestProviderConfig + testAccAssignmentPolicyConfig + `
resource "orcasecurity_admission_controller_policy_assignment" "test" {
  name              = "tf-acc-assignment-1-renamed"
  description       = "updated"
  full_organization = true
  policy_ids        = [orcasecurity_admission_controller_policy.for_assignment.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy_assignment.test", "name", "tf-acc-assignment-1-renamed"),
				),
			},
		},
	})
}

func TestAccAdmissionControllerPolicyAssignmentResource_RequiresScope(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_admission_controller_policy_assignment" "invalid" {
  name = "tf-acc-assignment-invalid"
}
`,
				ExpectError: regexp.MustCompile(`(?s)full_organization.*clusters.*cloud_accounts`),
			},
		},
	})
}
