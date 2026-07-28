package group_access_test

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const testAccGroupAccessResourceName = "orcasecurity_group_access.acc"

func testAccGroupAccessClient(t *testing.T) *api_client.APIClient {
	t.Helper()
	endpoint := os.Getenv("ORCASECURITY_API_ENDPOINT")
	token := os.Getenv("ORCASECURITY_API_TOKEN")
	c, err := api_client.NewAPIClient(&endpoint, &token)
	if err != nil {
		t.Fatalf("build api client: %s", err)
	}
	return c
}

func testAccCheckGroupAccessDestroyed(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[testAccGroupAccessResourceName]
		if !ok {
			return nil // never created
		}
		id := rs.Primary.ID
		groupID := rs.Primary.Attributes["group_id"]
		list, err := testAccGroupAccessClient(t).ListGroupAccessForGroup(groupID)
		if err != nil {
			return fmt.Errorf("verifying destroy: %s", err)
		}
		for _, ga := range list {
			if ga.ID == id {
				return fmt.Errorf("group access %s still exists after destroy", id)
			}
		}
		return nil
	}
}

func TestAccGroupAccess_envIDs(t *testing.T) {
	gid := os.Getenv("ORCASECURITY_ACC_GROUP_ACCESS_GROUP_ID")
	rid := os.Getenv("ORCASECURITY_ACC_GROUP_ACCESS_ROLE_ID")
	fid := os.Getenv("ORCASECURITY_ACC_GROUP_ACCESS_USER_FILTER_ID")
	if gid == "" || rid == "" || fid == "" {
		t.Skip("Skipping: set ORCASECURITY_ACC_GROUP_ACCESS_GROUP_ID, ORCASECURITY_ACC_GROUP_ACCESS_ROLE_ID, and ORCASECURITY_ACC_GROUP_ACCESS_USER_FILTER_ID to run this acceptance test")
	}

	cfg := func(filterID string) string {
		return fmt.Sprintf(`
%s
resource "orcasecurity_group_access" "acc" {
  group_id           = %q
  role_id            = %q
  all_cloud_accounts = false
  user_filters       = [%q]
}
`, orcasecurity.TestProviderConfig, gid, rid, filterID)
	}

	steps := []resource.TestStep{
		{
			Config: cfg(fid),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(testAccGroupAccessResourceName, "group_id", gid),
				resource.TestCheckResourceAttr(testAccGroupAccessResourceName, "role_id", rid),
				resource.TestCheckResourceAttr(testAccGroupAccessResourceName, "user_filters.#", "1"),
				resource.TestCheckResourceAttr(testAccGroupAccessResourceName, "user_filters.0", fid),
				resource.TestCheckResourceAttrSet(testAccGroupAccessResourceName, "id"),
			),
		},
		{
			ResourceName:      testAccGroupAccessResourceName,
			ImportState:       true,
			ImportStateVerify: true,
		},
	}

	// Optional update step; set ORCASECURITY_ACC_GROUP_ACCESS_USER_FILTER_ID_2.
	if fid2 := os.Getenv("ORCASECURITY_ACC_GROUP_ACCESS_USER_FILTER_ID_2"); fid2 != "" {
		steps = append(steps, resource.TestStep{
			Config: cfg(fid2),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(testAccGroupAccessResourceName, "user_filters.#", "1"),
				resource.TestCheckResourceAttr(testAccGroupAccessResourceName, "user_filters.0", fid2),
				resource.TestCheckResourceAttrSet(testAccGroupAccessResourceName, "id"),
			),
		})
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGroupAccessDestroyed(t),
		Steps:                    steps,
	})
}
