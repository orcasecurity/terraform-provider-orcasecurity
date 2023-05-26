package sonar_test

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccDataSourceConfig = orcasecurity.TestProviderConfig + `
data "orcasecurity_discovery_results" "test" {
	query = jsonencode({
	  "type" : "object_set",
	  "models" : ["Inventory"],
	  "keys" : ["Inventory"],
	  "with" : {
		"type" : "operation",
		"values" : [
		  { "type" : "str",
			"key" : "Name",
			"values" : ["4d9b3e13-22dd-4861-8b2d-4c8939cb599e"],
			"operator" : "in"
		  }
		],
		"operator" : "and"
	  }
	})
	limit          = 2
	start_at_index = 0
  }
`

func TestAccSonarQueryDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.orcasecurity_discovery_results.test", "name"),
					resource.TestCheckResourceAttrSet("data.orcasecurity_discovery_results.test", "type"),
					resource.TestCheckResourceAttrSet("data.orcasecurity_discovery_results.test", "data"),
					resource.TestCheckResourceAttrWith("data.orcasecurity_discovery_results.test", "id", func(value string) error {
						// it must be a valid UUID
						_, err := uuid.Parse(value)
						return err
					}),
				),
			},
		},
	})
}
