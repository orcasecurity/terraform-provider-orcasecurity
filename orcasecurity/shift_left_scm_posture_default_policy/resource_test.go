package shift_left_scm_posture_default_policy_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccScmPostureDefaultPolicy_adopt(t *testing.T) {
	// The snapshot below runs before resource.Test can auto-skip, so gate on
	// TF_ACC explicitly (CI runs unit tests without credentials).
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to run acceptance tests")
	}

	// Snapshot the live singleton and restore it after the test: the policy
	// is org-wide and never deletable, so the applied config would otherwise
	// leak into the lab environment.
	orcasecurity.TestAccPreCheck(t)
	client := acctest.APIClient(t)
	original, err := client.GetScmPostureDefaultPolicy()
	if err != nil {
		t.Fatalf("failed to snapshot scm posture default policy: %s", err)
	}
	t.Cleanup(func() {
		var data api_client.ScmPostureDefaultPolicyData
		if len(original.PolicyData) > 0 {
			_ = json.Unmarshal(original.PolicyData, &data)
		}
		restore := api_client.ScmPostureDefaultPolicyWrite{
			Disabled:   original.Disabled,
			PolicyData: data,
		}
		if _, err := client.UpdateScmPostureDefaultPolicy(restore); err != nil {
			t.Errorf("failed to restore scm posture default policy: %s", err)
		}
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_scm_posture_default_policy" "t" {
  disabled = %t
}`, original.Disabled),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_scm_posture_default_policy.t", "id", original.ID),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_scm_posture_default_policy.t", "name", original.Name),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_scm_posture_default_policy.t", "disabled", fmt.Sprintf("%t", original.Disabled)),
				),
			},
			{
				ResourceName:      "orcasecurity_shift_left_scm_posture_default_policy.t",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     original.ID,
				ImportStateVerifyIgnore: []string{"controls"},
			},
		},
	})
}
