package shift_left_projects_test

import (
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccShiftLeftProjectsDataSource asserts the data source enumerates at
// least one shift-left project against the live API. Deferred: not run as
// part of this change; requires TF_ACC=1 and real credentials.
func TestAccShiftLeftProjectsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_shift_left_projects" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith("data.orcasecurity_shift_left_projects.all", "projects.#", func(v string) error {
						if v == "0" {
							return fmt.Errorf("expected at least one project, got %s", v)
						}
						return nil
					}),
				),
			},
		},
	})
}
