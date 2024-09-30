package shift_left_project_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccShiftLeftCveExceptionListResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_shift_left_project" "shift_left_project_1" {
  name             = "Project 1"
  description      = "Project for all repos"
  key              = "project-1"
  default_policies = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_project.shift_left_project_1", "name", "Project 1"),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_project.shift_left_project_1", "description", "Project for all repos"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_shift_left_project.shift_left_project_1",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_shift_left_project" "shift_left_project_1" {
  name             = "Project 2"
  description      = "Project 2 for all repos"
  key              = "project-1"
  default_policies = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_project.shift_left_project_1", "name", "Project 2"),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_project.shift_left_project_1", "description", "Project 2 for all repos"),
				),
			},
		},
	})
}
