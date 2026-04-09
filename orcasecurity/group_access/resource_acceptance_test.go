package group_access_test

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccGroupAccessResourceName = "orcasecurity_group_access.acc"

func TestAccGroupAccess_envIDs(t *testing.T) {
	gid := os.Getenv("ORCASECURITY_ACC_GROUP_ACCESS_GROUP_ID")
	rid := os.Getenv("ORCASECURITY_ACC_GROUP_ACCESS_ROLE_ID")
	fid := os.Getenv("ORCASECURITY_ACC_GROUP_ACCESS_USER_FILTER_ID")
	if gid == "" || rid == "" || fid == "" {
		t.Skip("Skipping: set ORCASECURITY_ACC_GROUP_ACCESS_GROUP_ID, ORCASECURITY_ACC_GROUP_ACCESS_ROLE_ID, and ORCASECURITY_ACC_GROUP_ACCESS_USER_FILTER_ID to run this acceptance test")
	}

	cfg := fmt.Sprintf(`
%s
resource "orcasecurity_group_access" "acc" {
  group_id             = %q
  role_id              = %q
  all_cloud_accounts   = false
  user_filters         = [%q]
}
`, orcasecurity.TestProviderConfig, gid, rid, fid)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testAccGroupAccessResourceName, "group_id", gid),
					resource.TestCheckResourceAttr(testAccGroupAccessResourceName, "role_id", rid),
					resource.TestCheckResourceAttr(testAccGroupAccessResourceName, "user_filters.#", "1"),
					resource.TestCheckResourceAttr(testAccGroupAccessResourceName, "user_filters.0", fid),
					resource.TestCheckResourceAttrSet(testAccGroupAccessResourceName, "id"),
				),
			},
		},
	})
}
