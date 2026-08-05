package shift_left_unit_test

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Import-only path: Read keeps legacy SCAN_ALL in state; apply migrates to SELECTED_REPOSITORIES.
func TestScmUnitApply_ImportedLegacyScanAllMigratesOnApply(t *testing.T) {
	stub := newSCMUnitStub()
	stub.unit["installation_mode"] = "SCAN_ALL"
	stub.start(t)

	config := stubConfig(`  configuration_settings = { pr_summary_appendix = "a" }`)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Import writes nothing; Read's SCAN_ALL lands in state.
				Config:             config,
				ResourceName:       stubResourceName,
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateId:      stubInstallationID + "/" + stubAccountSlug,
				Check: resource.TestCheckResourceAttr(stubResourceName,
					"installation_mode", "SCAN_ALL"),
			},
			{
				// Apply migrates to what the API stores.
				Config: config,
				Check: resource.TestCheckResourceAttr(stubResourceName,
					"installation_mode", "SELECTED_REPOSITORIES"),
			},
		},
	})
}

// Legacy SCAN_ALL in state must plan volatile attrs unknown or migration write causes inconsistent result after apply.
func TestScmUnitApply_ImportedLegacyScanAllWithVolatileSideEffects(t *testing.T) {
	stub := newSCMUnitStub()
	stub.volatileSideEffects = true
	stub.unit["installation_mode"] = "SCAN_ALL"
	stub.unit["integrated_repositories_count"] = 12
	stub.unit["scan_all_state"] = "RUNNING"
	stub.start(t)

	config := stubConfig(``)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             config,
				ResourceName:       stubResourceName,
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateId:      stubInstallationID + "/" + stubAccountSlug,
				Check: resource.TestCheckResourceAttr(stubResourceName,
					"scan_all_state", "RUNNING"),
			},
			{
				// Migration write halts enrollment; state must pick up moved volatile values.
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(stubResourceName, "installation_mode", "SELECTED_REPOSITORIES"),
					resource.TestCheckResourceAttr(stubResourceName, "scan_all_state", "STOPPED"),
					resource.TestCheckResourceAttr(stubResourceName, "integrated_repositories_count", "5"),
				),
			},
		},
	})
}
