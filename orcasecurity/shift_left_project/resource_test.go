package shift_left_project_test

import (
	"fmt"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccShiftLeftCveExceptionListResource_Basic(t *testing.T) {
	key := fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_project" "shift_left_project_1" {
  name             = "TF Acc %[1]s"
  description      = "Project for all repos"
  key              = %[1]q
  default_policies = true
}
`, key),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_project.shift_left_project_1", "name", "TF Acc "+key),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_project.shift_left_project_1", "description", "Project for all repos"),
				),
			},
			{
				ResourceName:      "orcasecurity_shift_left_project.shift_left_project_1",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_project" "shift_left_project_1" {
  name             = "TF Acc %[1]s v2"
  description      = "Project 2 for all repos"
  key              = %[1]q
  default_policies = true
}
`, key),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_project.shift_left_project_1", "name", "TF Acc "+key+" v2"),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_project.shift_left_project_1", "description", "Project 2 for all repos"),
				),
			},
		},
	})
}
