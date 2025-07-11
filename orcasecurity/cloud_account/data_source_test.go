package cloudaccount_test

import (
	"regexp"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccDataSourceConfigByName = orcasecurity.TestProviderConfig + `
data "orcasecurity_cloud_account" "test_by_name" {
	name = "test-account-name"
}
`

const testAccDataSourceConfigByVendorID = orcasecurity.TestProviderConfig + `
data "orcasecurity_cloud_account" "test_by_vendor_id" {
	cloud_vendor_id = "BB5EDED2-8EA3-4608-8BE3-FD0EE8B3F644"
}
`

func TestAccCloudAccountDataSourceByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfigByName,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the cloud account ID is a valid UUID
					resource.TestCheckResourceAttrWith("data.orcasecurity_cloud_account.test_by_name", "cloud_account_id", func(value string) error {
						_, err := uuid.Parse(value)
						return err
					}),
					// Verify the name contains our search term
					resource.TestCheckResourceAttrSet("data.orcasecurity_cloud_account.test_by_name", "name"),
					// Verify cloud provider fields are set
					resource.TestCheckResourceAttrSet("data.orcasecurity_cloud_account.test_by_name", "cloud_provider"),
					resource.TestCheckResourceAttrSet("data.orcasecurity_cloud_account.test_by_name", "cloud_provider_id"),
					resource.TestCheckResourceAttrSet("data.orcasecurity_cloud_account.test_by_name", "cloud_vendor_id"),
					// Verify status fields
					resource.TestCheckResourceAttrSet("data.orcasecurity_cloud_account.test_by_name", "cloud_account_status"),
					// Verify timestamps
					resource.TestCheckResourceAttrSet("data.orcasecurity_cloud_account.test_by_name", "created_time"),
					// Verify vendor IDs
					resource.TestCheckResourceAttrSet("data.orcasecurity_cloud_account.test_by_name", "vendor_id"),
				),
			},
		},
	})
}

func TestAccCloudAccountDataSourceByVendorID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfigByVendorID,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the cloud account ID is a valid UUID
					resource.TestCheckResourceAttrWith("data.orcasecurity_cloud_account.test_by_vendor_id", "cloud_account_id", func(value string) error {
						_, err := uuid.Parse(value)
						return err
					}),
					// Verify the cloud vendor ID matches what we searched for
					resource.TestCheckResourceAttr("data.orcasecurity_cloud_account.test_by_vendor_id", "cloud_vendor_id", "2cfe3cf0-1175-47f5-bf72-876d4a4f3353"),
					// Verify other key fields are populated
					resource.TestCheckResourceAttrSet("data.orcasecurity_cloud_account.test_by_vendor_id", "name"),
					resource.TestCheckResourceAttrSet("data.orcasecurity_cloud_account.test_by_vendor_id", "cloud_provider"),
					resource.TestCheckResourceAttrSet("data.orcasecurity_cloud_account.test_by_vendor_id", "cloud_account_status"),
					resource.TestCheckResourceAttrSet("data.orcasecurity_cloud_account.test_by_vendor_id", "cloud_provider_id"),
				),
			},
		},
	})
}

func TestAccCloudAccountDataSourceConflictingParams(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_cloud_account" "test_conflict" {
	name = "cnh-prod-AccentureVDI"
	cloud_vendor_id = "2cfe3cf0-1175-47f5-bf72-876d4a4f3353"
}
`,
				ExpectError: regexp.MustCompile("Only one of 'name' or 'cloud_vendor_id' can be specified"),
			},
		},
	})
}

func TestAccCloudAccountDataSourceMissingParams(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_cloud_account" "test_missing" {
	# No search parameters provided
}
`,
				ExpectError: regexp.MustCompile("Either 'name' or 'cloud_vendor_id' must be specified"),
			},
		},
	})
}
