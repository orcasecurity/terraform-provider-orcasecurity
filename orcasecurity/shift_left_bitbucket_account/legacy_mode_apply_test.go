package shift_left_bitbucket_account_test

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Read reports installation_mode verbatim, so a unit still on the legacy SCAN_ALL
// keeps that value instead of being displayed as SELECTED_REPOSITORIES. Import is the
// only path that puts it in state — create and update both write first, and the write
// remaps SCAN_ALL before it can persist — so this is the case that exercises it.
//
// installation_mode is Optional+Computed, so a config that omits it carries the state
// value into the plan. Carrying SCAN_ALL forward unchanged would plan a value the write
// path never sends, and the apply would fail with "inconsistent result after apply".
func TestScmUnitApply_ImportedLegacyScanAllMigratesOnApply(t *testing.T) {
	stub := newSCMUnitStub()
	stub.unit["installation_mode"] = "SCAN_ALL"
	stub.start(t)

	config := stubConfig(`  configuration_settings = { pr_summary_appendix = "a" }`)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Import writes nothing, so Read's SCAN_ALL is what lands in state.
				Config:             config,
				ResourceName:       stubResourceName,
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateId:      stubInstallationID + "/" + stubAccountSlug,
				Check: resource.TestCheckResourceAttr(stubResourceName,
					"installation_mode", "SCAN_ALL"),
			},
			{
				// The apply migrates it to what the API actually stores.
				Config: config,
				Check: resource.TestCheckResourceAttr(stubResourceName,
					"installation_mode", "SELECTED_REPOSITORIES"),
			},
		},
	})
}
