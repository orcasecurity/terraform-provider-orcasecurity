package compliance_framework_selection_test

import (
	"fmt"
	"os"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccComplianceFrameworkSelection_Basic flips a disposable framework's
// selection. Last step sets scopes = [] so destroy (state-only) leaves it
// disabled. Unset ORCA_TEST_COMPLIANCE_FRAMEWORK_ID skips; the organization
// scope is shared tenant state — point this at an inactive framework.
func TestAccComplianceFrameworkSelection_Basic(t *testing.T) {
	frameworkID := os.Getenv("ORCA_TEST_COMPLIANCE_FRAMEWORK_ID")
	if frameworkID == "" {
		t.Skip("ORCA_TEST_COMPLIANCE_FRAMEWORK_ID not set; need a framework this test may flip")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_compliance_framework_selection" "test" {
  framework_id = %q
  scopes       = ["user"]
}
`, frameworkID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_compliance_framework_selection.test", "framework_id", frameworkID),
					resource.TestCheckResourceAttr("orcasecurity_compliance_framework_selection.test", "id", frameworkID),
					resource.TestCheckResourceAttr("orcasecurity_compliance_framework_selection.test", "scopes.#", "1"),
				),
			},
			{
				ResourceName:      "orcasecurity_compliance_framework_selection.test",
				ImportState:       true,
				ImportStateId:     frameworkID,
				ImportStateVerify: true,
			},
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_compliance_framework_selection" "test" {
  framework_id = %q
  scopes       = []
}
`, frameworkID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_compliance_framework_selection.test", "scopes.#", "0"),
				),
			},
		},
	})
}
