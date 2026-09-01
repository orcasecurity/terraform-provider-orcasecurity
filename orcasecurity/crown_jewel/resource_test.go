package crown_jewel_test

import (
	"fmt"
	"os"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ORCA_TEST_CROWN_JEWEL_GROUP_UNIQUE_ID must be a real inventory group_unique_id
// in the lab tenant. Fabricated ids can POST successfully but miss the
// read-your-writes GET when the list shape/filtering changes; use a disposable
// asset that is safe to mark and unmark.
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
