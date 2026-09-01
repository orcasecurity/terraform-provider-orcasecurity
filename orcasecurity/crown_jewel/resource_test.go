package crown_jewel_test

import (
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCrownJewelResource_Basic(t *testing.T) {
	groupID := fmt.Sprintf("tf-acc-crown-jewel-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_crown_jewel" "test" {
  group_unique_id = %q
  description     = "tf-acc crown jewel"
}
`, groupID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_crown_jewel.test", "group_unique_id", groupID),
					resource.TestCheckResourceAttr("orcasecurity_crown_jewel.test", "description", "tf-acc crown jewel"),
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
  description     = "tf-acc crown jewel updated"
}
`, groupID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_crown_jewel.test", "description", "tf-acc crown jewel updated"),
					resource.TestCheckResourceAttr("orcasecurity_crown_jewel.test", "id", groupID),
				),
			},
		},
	})
}
