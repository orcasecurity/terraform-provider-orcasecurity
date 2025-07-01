package business_unit_test

import (
	"regexp"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccBusinessUnitDataSourceByNameConfig = orcasecurity.TestProviderConfig + `
data "orcasecurity_business_unit" "test" {
	name = "AWS FootPrint"
}
`

const testAccBusinessUnitDataSourceByIDConfig = orcasecurity.TestProviderConfig + `
data "orcasecurity_business_unit" "test" {
	id = "68671f6f-8f63-4cc3-a842-f7d034078955"
}
`

const testAccBusinessUnitDataSourceConflictingConfig = orcasecurity.TestProviderConfig + `
data "orcasecurity_business_unit" "test" {
	id   = "68671f6f-8f63-4cc3-a842-f7d034078955"
	name = "AWS FootPrint"
}
`

const testAccBusinessUnitDataSourceMissingConfig = orcasecurity.TestProviderConfig + `
data "orcasecurity_business_unit" "test" {
	# Neither id nor name specified
}
`

func TestAccBusinessUnitDataSourceByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBusinessUnitDataSourceByNameConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.orcasecurity_business_unit.test", "name", "AWS FootPrint"),
					resource.TestCheckResourceAttrWith("data.orcasecurity_business_unit.test", "id", func(value string) error {
						// it must be a valid UUID
						_, err := uuid.Parse(value)
						return err
					}),
					// Check object element counts instead of the objects themselves
					resource.TestCheckResourceAttrSet("data.orcasecurity_business_unit.test", "filter.%"),
					resource.TestCheckResourceAttrSet("data.orcasecurity_business_unit.test", "shift_left_filter.%"),
					// Check set element counts using .# syntax
					resource.TestCheckResourceAttrSet("data.orcasecurity_business_unit.test", "filter.cloud_providers.#"),
					resource.TestCheckResourceAttrSet("data.orcasecurity_business_unit.test", "shift_left_filter.shift_left_projects.#"),
				),
			},
		},
	})
}

func TestAccBusinessUnitDataSourceByID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBusinessUnitDataSourceByIDConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.orcasecurity_business_unit.test", "id", "68671f6f-8f63-4cc3-a842-f7d034078955"),
					resource.TestCheckResourceAttr("data.orcasecurity_business_unit.test", "name", "AWS FootPrint"),
					// Check object element counts instead of the objects themselves
					resource.TestCheckResourceAttrSet("data.orcasecurity_business_unit.test", "filter.%"),
					resource.TestCheckResourceAttrSet("data.orcasecurity_business_unit.test", "shift_left_filter.%"),
					// Check set element counts using .# syntax
					resource.TestCheckResourceAttrSet("data.orcasecurity_business_unit.test", "filter.cloud_providers.#"),
					resource.TestCheckResourceAttrSet("data.orcasecurity_business_unit.test", "shift_left_filter.shift_left_projects.#"),
				),
			},
		},
	})
}

func TestAccBusinessUnitDataSourceConflictingParams(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccBusinessUnitDataSourceConflictingConfig,
				ExpectError: regexp.MustCompile("Cannot specify both 'id' and 'name'"),
			},
		},
	})
}

func TestAccBusinessUnitDataSourceMissingParams(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccBusinessUnitDataSourceMissingConfig,
				ExpectError: regexp.MustCompile("Must specify either 'id' or 'name'"),
			},
		},
	})
}

func TestAccBusinessUnitDataSourceNonExistentName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_business_unit" "test" {
	name = "NonExistentBusinessUnit"
}
`,
				ExpectError: regexp.MustCompile("business unit with name 'NonExistentBusinessUnit' does not exist"),
			},
		},
	})
}

func TestAccBusinessUnitDataSourceNonExistentID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_business_unit" "test" {
	id = "00000000-0000-0000-0000-000000000000"
}
`,
				ExpectError: regexp.MustCompile("Unable to read business unit"),
			},
		},
	})
}
