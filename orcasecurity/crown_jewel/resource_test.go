package crown_jewel_test

import (
	"fmt"
	"os"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ORCA_TEST_CROWN_JEWEL_GROUP_UNIQUE_ID must be a real inventory group_unique_id
// in the lab tenant that is not already user-marked or Orca-detected. Create
// refuses unknown ids, already-marked assets (import those instead), and
// engine-managed detections. Destroy leaves an is_crown_jewel=false override,
// not a hard delete.
func TestAccCrownJewelResource_Basic(t *testing.T) {
	groupID := os.Getenv("ORCA_TEST_CROWN_JEWEL_GROUP_UNIQUE_ID")
	if groupID == "" {
		t.Skip("ORCA_TEST_CROWN_JEWEL_GROUP_UNIQUE_ID not set; need a real inventory group_unique_id")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_crown_jewel" "test" {
  group_unique_id = %q
  description     = "Customer data"
}
`, groupID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_crown_jewel.test", "group_unique_id", groupID),
					resource.TestCheckResourceAttr("orcasecurity_crown_jewel.test", "description", "Customer data"),
					resource.TestCheckResourceAttr("orcasecurity_crown_jewel.test", "id", groupID),
				),
			},
			{
				ResourceName:      "orcasecurity_crown_jewel.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_crown_jewel" "test" {
  group_unique_id = %q
  description     = "Critical business function"
}
`, groupID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_crown_jewel.test", "description", "Critical business function"),
					resource.TestCheckResourceAttr("orcasecurity_crown_jewel.test", "id", groupID),
				),
			},
		},
	})
}

func TestAccCrownJewelDataSource_Basic(t *testing.T) {
	groupID := os.Getenv("ORCA_TEST_CROWN_JEWEL_GROUP_UNIQUE_ID")
	if groupID == "" {
		t.Skip("ORCA_TEST_CROWN_JEWEL_GROUP_UNIQUE_ID not set; need a real inventory group_unique_id")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_crown_jewel" "test" {
  group_unique_id = %q
  description     = "Customer data"
}

data "orcasecurity_crown_jewel" "test" {
  group_unique_id = orcasecurity_crown_jewel.test.group_unique_id
}
`, groupID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.orcasecurity_crown_jewel.test", "group_unique_id", groupID),
					resource.TestCheckResourceAttr("data.orcasecurity_crown_jewel.test", "description", "Customer data"),
					resource.TestCheckResourceAttr("data.orcasecurity_crown_jewel.test", "id", groupID),
				),
			},
		},
	})
}
