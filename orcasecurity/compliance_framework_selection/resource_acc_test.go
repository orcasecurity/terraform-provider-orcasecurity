package compliance_framework_selection_test

import (
	"fmt"
	"os"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccComplianceFrameworkSelection_Basic flips a built-in framework's
// selection under restore_on_destroy so destroy puts the pre-adoption scopes
// back. Unset ORCA_TEST_COMPLIANCE_FRAMEWORK_ID skips; the organization scope
// is shared tenant state.
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
  framework_id       = %q
  scopes             = ["user"]
  restore_on_destroy = true
}
`, frameworkID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_compliance_framework_selection.test", "framework_id", frameworkID),
					resource.TestCheckResourceAttr("orcasecurity_compliance_framework_selection.test", "id", frameworkID),
					resource.TestCheckResourceAttr("orcasecurity_compliance_framework_selection.test", "active", "true"),
					resource.TestCheckResourceAttr("orcasecurity_compliance_framework_selection.test", "scopes.#", "1"),
				),
			},
			{
				ResourceName:            "orcasecurity_compliance_framework_selection.test",
				ImportState:             true,
				ImportStateId:           frameworkID,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"restore_on_destroy", "original_scopes"},
			},
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_compliance_framework_selection" "test" {
  framework_id       = %q
  scopes             = []
  restore_on_destroy = true
}
`, frameworkID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_compliance_framework_selection.test", "scopes.#", "0"),
					resource.TestCheckResourceAttr("orcasecurity_compliance_framework_selection.test", "active", "false"),
				),
			},
		},
	})
}
