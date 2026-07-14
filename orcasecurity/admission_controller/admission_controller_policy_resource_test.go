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
			// Update: rename + flip enforcement + deactivate + add description.
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
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy.test", "description", "updated"),
				),
			},
			// Clear description: removing the attribute must clear it remotely
			// and converge (the client sends an explicit null on PUT).
			{
				Config: orcasecurity.TestProviderConfig + testAccPolicyControlConfig + `
resource "orcasecurity_admission_controller_policy" "test" {
  name               = "tf-acc-policy-1-cleared"
  is_active          = false
  enforcement_action = "block"
  controls           = [orcasecurity_admission_controller_control.for_policy.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("orcasecurity_admission_controller_policy.test", "description"),
				),
			},
		},
	})
}

// Exercises controls set membership changes: create with two controls, then
// drop one. All other policy tests use exactly one control, so this is the
// only coverage of an actual set add/remove against the API.
func TestAccAdmissionControllerPolicyResource_RemoveControl(t *testing.T) {
	const twoControlsConfig = `
data "orcasecurity_admission_controller_template" "repos_multi" {
  name = "k8sallowedrepos"
}

data "orcasecurity_admission_controller_template" "tty_multi" {
  name = "k8sdisallowinteractivetty"
}

resource "orcasecurity_admission_controller_control" "multi_a" {
  name        = "tf-acc-policy-multi-control-a"
  template_id = data.orcasecurity_admission_controller_template.repos_multi.id
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

resource "orcasecurity_admission_controller_control" "multi_b" {
  name        = "tf-acc-policy-multi-control-b"
  template_id = data.orcasecurity_admission_controller_template.tty_multi.id
  cluster_scope = {
    kinds = [
      {
        kinds      = ["Pod"]
        api_groups = [""]
        versions   = [""]
      }
    ]
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + twoControlsConfig + `
resource "orcasecurity_admission_controller_policy" "multi" {
  name = "tf-acc-policy-multi"
  controls = [
    orcasecurity_admission_controller_control.multi_a.id,
    orcasecurity_admission_controller_control.multi_b.id,
  ]
}
`,
				Check: resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy.multi", "controls.#", "2"),
			},
			// Remove one control; the other must stay attached.
			{
				Config: orcasecurity.TestProviderConfig + twoControlsConfig + `
resource "orcasecurity_admission_controller_policy" "multi" {
  name     = "tf-acc-policy-multi"
  controls = [orcasecurity_admission_controller_control.multi_a.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy.multi", "controls.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(
						"orcasecurity_admission_controller_policy.multi", "controls.*",
						"orcasecurity_admission_controller_control.multi_a", "id",
					),
				),
			},
		},
	})
}

// The most common update — change a field without renaming — is rejected by
// the backend's PUT route: its name-uniqueness check does not exclude the
// policy itself ("Policy with name 'X' already exists."). The client works
// around it by updating via PATCH with the full payload (same route the Orca
// UI uses); this test pins that the workaround keeps working.
func TestAccAdmissionControllerPolicyResource_UpdateWithoutRename(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + testAccPolicyControlConfig + `
resource "orcasecurity_admission_controller_policy" "test" {
  name     = "tf-acc-policy-norename"
  controls = [orcasecurity_admission_controller_control.for_policy.id]
}
`,
				Check: resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy.test", "is_active", "true"),
			},
			// Same name, different field values.
			{
				Config: orcasecurity.TestProviderConfig + testAccPolicyControlConfig + `
resource "orcasecurity_admission_controller_policy" "test" {
  name               = "tf-acc-policy-norename"
  is_active          = false
  enforcement_action = "block"
  controls           = [orcasecurity_admission_controller_control.for_policy.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy.test", "is_active", "false"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy.test", "enforcement_action", "block"),
				),
			},
		},
	})
}
