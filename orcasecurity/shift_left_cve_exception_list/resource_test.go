package shift_left_cve_exception_list_test

import (
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccShiftLeftCveExceptionListResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_shift_left_cve_exception_list" "shiftleft_exception_list_1" {
  name        = "Exception List with Terraform"
  description = "Log4Shell Exception List"
  disabled    = false
  vulnerabilities = [
    {
      cve_id      = "cve-2021-44228"
      description = "log4shell"
      disabled    = false
      expiration  = "2024/09/25"
    }
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_cve_exception_list.shiftleft_exception_list_1", "name", "Exception List with Terraform"),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_cve_exception_list.shiftleft_exception_list_1", "description", "Log4Shell Exception List"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_shift_left_cve_exception_list.shiftleft_exception_list_1",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_shift_left_cve_exception_list" "shiftleft_exception_list_1" {
  name        = "Exception List with Terraform 2"
  description = "Log4Shell Exception List 2"
  disabled    = false
  vulnerabilities = [
    {
      cve_id      = "cve-2021-44228"
      description = "log4shell"
      disabled    = false
      expiration  = "2024/09/25"
    }
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_cve_exception_list.shiftleft_exception_list_1", "name", "Exception List with Terraform 2"),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_cve_exception_list.shiftleft_exception_list_1", "description", "Log4Shell Exception List 2"),
				),
			},
		},
	})
}

// Behind the org flag shiftleft_cve_exception_list_disabled_semantics, list-level disabled=true is
// stored as "every entry disabled" and omitted from responses; either way the plan must stay clean.
func TestAccShiftLeftCveExceptionListResource_ListLevelDisabled(t *testing.T) {
	config := func(entryDisabled bool) string {
		return orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_cve_exception_list" "disabled" {
  name     = "tf-acc-cve-list-disabled"
  disabled = true
  vulnerabilities = [
    {
      cve_id      = "cve-2021-44228"
      description = "log4shell"
      disabled    = %t
    }
  ]
}
`, entryDisabled)
	}
	address := "orcasecurity_shift_left_cve_exception_list.disabled"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config(false), Check: resource.TestCheckResourceAttr(address, "disabled", "true")},
			{Config: config(true), Check: resource.TestCheckResourceAttr(address, "vulnerabilities.0.disabled", "true")},
			{ResourceName: address, ImportState: true, ImportStateVerify: true},
		},
	})
}
