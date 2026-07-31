package user_access_test

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const testAccUserAccessResourceName = "orcasecurity_user_access.acc"

func testAccUserAccessClient(t *testing.T) *api_client.APIClient {
	t.Helper()
	endpoint := os.Getenv("ORCASECURITY_API_ENDPOINT")
	token := os.Getenv("ORCASECURITY_API_TOKEN")
	c, err := api_client.NewAPIClient(&endpoint, &token)
	if err != nil {
		t.Fatalf("build api client: %s", err)
	}
	return c
}

func testAccCheckUserAccessDestroyed(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[testAccUserAccessResourceName]
		if !ok {
			return nil // never created
		}
		id := rs.Primary.ID
		userID := rs.Primary.Attributes["user_id"]
		list, err := testAccUserAccessClient(t).ListUserAccessForUser(userID)
		if err != nil {
			return fmt.Errorf("verifying destroy: %s", err)
		}
		for _, ua := range list {
			if ua.ID == id {
				return fmt.Errorf("user access %s still exists after destroy", id)
			}
		}
		return nil
	}
}

func TestAccUserAccess_envIDs(t *testing.T) {
	uid := os.Getenv("ORCASECURITY_ACC_USER_ACCESS_USER_ID")
	rid := os.Getenv("ORCASECURITY_ACC_USER_ACCESS_ROLE_ID")
	fid := os.Getenv("ORCASECURITY_ACC_USER_ACCESS_USER_FILTER_ID")
	if uid == "" || rid == "" || fid == "" {
		t.Skip("Skipping: set ORCASECURITY_ACC_USER_ACCESS_USER_ID, ORCASECURITY_ACC_USER_ACCESS_ROLE_ID, and ORCASECURITY_ACC_USER_ACCESS_USER_FILTER_ID to run this acceptance test")
	}

	cfg := func(filterID string) string {
		return fmt.Sprintf(`
%s
resource "orcasecurity_user_access" "acc" {
  user_id            = %q
  role_id            = %q
  all_cloud_accounts = false
  user_filters       = [%q]
}
`, orcasecurity.TestProviderConfig, uid, rid, filterID)
	}

	steps := []resource.TestStep{
		{
			Config: cfg(fid),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(testAccUserAccessResourceName, "user_id", uid),
				resource.TestCheckResourceAttr(testAccUserAccessResourceName, "role_id", rid),
				resource.TestCheckResourceAttr(testAccUserAccessResourceName, "user_filters.#", "1"),
				resource.TestCheckResourceAttr(testAccUserAccessResourceName, "user_filters.0", fid),
				resource.TestCheckResourceAttrSet(testAccUserAccessResourceName, "id"),
			),
		},
		{
			ResourceName:      testAccUserAccessResourceName,
			ImportState:       true,
			ImportStateVerify: true,
		},
	}

	// Optional update step; set ORCASECURITY_ACC_USER_ACCESS_USER_FILTER_ID_2.
	if fid2 := os.Getenv("ORCASECURITY_ACC_USER_ACCESS_USER_FILTER_ID_2"); fid2 != "" {
		steps = append(steps, resource.TestStep{
			Config: cfg(fid2),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(testAccUserAccessResourceName, "user_filters.#", "1"),
				resource.TestCheckResourceAttr(testAccUserAccessResourceName, "user_filters.0", fid2),
				resource.TestCheckResourceAttrSet(testAccUserAccessResourceName, "id"),
			),
		})
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserAccessDestroyed(t),
		Steps:                    steps,
	})
}
