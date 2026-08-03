package shift_left_bitbucket_account_test

import (
	"fmt"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// These cases pin the volatile-attribute contract at the Terraform plan/apply level.
// The unit tests in shift_left_integration cover settleVolatileAttrs in isolation,
// but the two failure modes below only appear once Terraform drives a real
// plan → apply → re-plan cycle, so they cannot be reproduced from a unit test:
//
//   - carrying a stale value forward across a write that moves it fails the apply
//     with "inconsistent result after apply";
//   - treating an unknown writable as "no write" carries the volatile value
//     forward as known and then replans it unknown on the apply re-plan once the
//     reference resolves, which Terraform rejects as "inconsistent final plan".

// newVolatileStub is the shared stub with server-side side effects switched on.
func newVolatileStub() *scmUnitStub {
	s := newSCMUnitStub()
	s.volatileSideEffects = true
	s.unit["integrated_repositories_count"] = 0
	s.unit["scan_all_state"] = "IDLE"
	s.unit["scm_posture_policy_id"] = "policy-a"
	return s
}

// Switching to scan-all enrols repositories, so integrated_repositories_count and
// scan_all_state both move as a side effect of the write. Carrying the prior values
// forward (plain UseStateForUnknown) fails the apply.
func TestScmUnitApply_VolatileAttributesMoveDuringApply(t *testing.T) {
	newVolatileStub().start(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: stubConfig(`  installation_mode = "SELECTED_REPOSITORIES"`),
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

// The volatile modifier must still let an unchanged configuration settle: re-applying
// an identical config has to leave no diff. resource.Test fails the step otherwise.
func TestScmUnitApply_VolatileAttributesSettleOnNoOpReapply(t *testing.T) {
	newVolatileStub().start(t)
	config := stubConfig(`  installation_mode = "SELECTED_REPOSITORIES"`)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{Config: config},
		},
	})
}

// A writable attribute that is unknown at plan time (bound to another resource) must
// not be read as "no write". Deciding from Plan.Raw treats the unknown as a match,
// carries the volatile value forward as known, and then plans it unknown on the apply
// re-plan once the reference resolves — which Terraform rejects.
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
  default_policies = false
  project_id       = terraform_data.project.output
}
`, projectInput, stubInstallationID, stubAccountSlug)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		// The unknown value is driven through terraform_data, which the builtin
		// provider only ships from Terraform 1.4. The modifier logic itself is
		// version-independent and unit-tested in shift_left_integration.
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_4_0),
		},
		Steps: []resource.TestStep{
			{Config: config("proj-1")},
			{Config: config("proj-2")},
		},
	})
}

// Control for the case above: the same project rebind written as a literal, so nothing
// in the plan is unknown. It isolates the cross-resource reference as the trigger.
func TestScmUnitApply_VolatileAttributesWithLiteralProjectRebind(t *testing.T) {
	newVolatileStub().start(t)
	config := func(projectID string) string {
		return stubConfig(fmt.Sprintf(`  default_policies = false
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
