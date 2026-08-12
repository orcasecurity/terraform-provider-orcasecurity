package shift_left_unit_test

import (
	"fmt"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// Plan/apply cycles that expose volatile-attribute failures unit tests cannot reproduce.

// newVolatileStub is the shared stub with server-side side effects switched on.
func newVolatileStub() *scmUnitStub {
	s := newSCMUnitStub()
	s.volatileSideEffects = true
	s.unit["integrated_repositories_count"] = 0
	s.unit["scan_all_state"] = "IDLE"
	s.unit["scm_posture_policy_id"] = "policy-a"
	return s
}

// Scan-all write moves integrated_repositories_count and scan_all_state; stale carry-forward fails apply.
func TestScmUnitApply_VolatileAttributesMoveDuringApply(t *testing.T) {
	newVolatileStub().start(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: stubConfig(`  adopt_existing     = true
  installation_mode = "SELECTED_REPOSITORIES"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(stubResourceName, "integrated_repositories_count", "0"),
					resource.TestCheckResourceAttr(stubResourceName, "scan_all_state", "IDLE"),
				),
			},
			{
				Config: stubConfig(`  installation_mode = "SCAN_ALL_INCLUDE_FUTURE"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(stubResourceName, "integrated_repositories_count", "37"),
					resource.TestCheckResourceAttr(stubResourceName, "scan_all_state", "RUNNING"),
				),
			},
		},
	})
}

// Identical config re-apply must leave no diff.
func TestScmUnitApply_VolatileAttributesSettleOnNoOpReapply(t *testing.T) {
	newVolatileStub().start(t)
	config := stubConfig(`  adopt_existing     = true
  installation_mode = "SELECTED_REPOSITORIES"`)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{Config: config},
		},
	})
}

// Unknown writable at plan time must not be treated as no-write or apply re-plan rejects inconsistent final plan.
func TestScmUnitApply_VolatileAttributesWithUnknownWritable(t *testing.T) {
	newVolatileStub().start(t)
	config := func(projectInput string) string {
		return orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "terraform_data" "project" {
  input = %q
}

resource "orcasecurity_shift_left_bitbucket_account" "test" {
  installation_id  = %q
  account_id       = %q
  adopt_existing   = true
  default_policies = false
  project_id       = terraform_data.project.output
}
`, projectInput, stubInstallationID, stubAccountSlug)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		// terraform_data drives unknown project_id; requires Terraform 1.4+.
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_4_0),
		},
		Steps: []resource.TestStep{
			{Config: config("proj-1")},
			{Config: config("proj-2")},
		},
	})
}

// Control: literal project rebind isolates cross-resource unknown as the trigger.
func TestScmUnitApply_VolatileAttributesWithLiteralProjectRebind(t *testing.T) {
	newVolatileStub().start(t)
	config := func(projectID string) string {
		return stubConfig(fmt.Sprintf(`  adopt_existing   = true
  default_policies = false
  project_id       = %q`, projectID))
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("proj-1")},
			{Config: config("proj-2")},
		},
	})
}
