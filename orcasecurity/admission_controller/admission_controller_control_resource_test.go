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
			// Clear description without renaming: removing the attribute must
			// clear it remotely and converge (the client sends an explicit
			// null on PUT; an omitted key would retain the old value).
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_admission_controller_template" "repos" {
  name = "k8sallowedrepos"
}

resource "orcasecurity_admission_controller_control" "test" {
  name        = "tf-acc-control-1-renamed"
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
					resource.TestCheckNoResourceAttr("orcasecurity_admission_controller_control.test", "description"),
				),
			},
			// Destroy happens automatically at the end of the test.
		},
	})
}

// Clearing input_parameters and the nested api_groups/versions on update:
// the PUT route retains an omitted input_parameters key, so this path relies
// on the client always serializing it ({} when unset). Uses a template
// without required parameters (clearing them on k8sallowedrepos would be
// rejected by the template's schema validation). cluster_scope is replaced
// wholesale by the backend, so dropped nested keys must converge too.
func TestAccAdmissionControllerControlResource_ClearOptionalAttributes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_admission_controller_template" "tty" {
  name = "k8sdisallowinteractivetty"
}

resource "orcasecurity_admission_controller_control" "clear" {
  name        = "tf-acc-control-clear"
  template_id = data.orcasecurity_admission_controller_template.tty.id
  cluster_scope = {
    kinds = [
      {
        kinds      = ["Pod"]
        api_groups = [""]
        versions   = [""]
      }
    ]
  }
  input_parameters = jsonencode({ exemptContainers = ["debug"] })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("orcasecurity_admission_controller_control.clear", "id"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_control.clear", "cluster_scope.kinds.0.api_groups.#", "1"),
				),
			},
			// Remove input_parameters, api_groups and versions: all must clear
			// remotely and converge (no refresh drift).
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_admission_controller_template" "tty" {
  name = "k8sdisallowinteractivetty"
}

resource "orcasecurity_admission_controller_control" "clear" {
  name        = "tf-acc-control-clear"
  template_id = data.orcasecurity_admission_controller_template.tty.id
  cluster_scope = {
    kinds = [
      {
        kinds = ["Pod"]
      }
    ]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("orcasecurity_admission_controller_control.clear", "input_parameters"),
					resource.TestCheckNoResourceAttr("orcasecurity_admission_controller_control.clear", "cluster_scope.kinds.0.api_groups"),
					resource.TestCheckNoResourceAttr("orcasecurity_admission_controller_control.clear", "cluster_scope.kinds.0.versions"),
				),
			},
		},
	})
}
