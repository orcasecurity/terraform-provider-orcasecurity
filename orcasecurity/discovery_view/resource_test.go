package discovery_view_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDiscoveryViewResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_discovery_view" "tf-disco-view-1" {
    name = "orca-disco-view-1"

    organization_level = true
    view_type = "discovery"
    extra_params = {}
    filter_data = {
        query = jsonencode({
            "models": [
                "AwsS3Bucket"
            ],
            "type": "object_set"
        })
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-view-1", "name", "orca-disco-view-1"),
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-view-1", "organizational_level", "true"),
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-view-1", "view_type", "discovery"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_discovery_view.tf-disco-view-1",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_discovery_view" "tf-disco-view-1" {
	name = "orca-disco-view-2"

	organization_level = false
    view_type = "discovery"
    extra_params = {}
    filter_data = {
        query = jsonencode({
            "models": [
                "AzureAcrImage"
            ],
            "type": "object_set"
        })
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-view-1", "name", "orca-disco-view-2"),
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-view-1", "organizational_level", "false"),
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-view-1", "view_type", "discovery"),
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-view-1", "filter_data.query", "jsonencode({\"models\": [\"AzureAcrImage\"],\"type\": \"object_set\"})"),
				),
			},
		},
	})
}

func TestAccDiscoveryViewResource_Description(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create with a description
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_discovery_view" "tf-disco-view-desc" {
    name               = "orca-disco-view-desc"
    description        = "initial description"
    organization_level = true
    view_type          = "discovery"
    extra_params       = {}
    filter_data = {
        query = jsonencode({ "models" : ["AwsS3Bucket"], "type" : "object_set" })
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-view-desc", "description", "initial description"),
				),
			},
			// import — description must round-trip from extra_params.description
			{
				ResourceName:      "orcasecurity_discovery_view.tf-disco-view-desc",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update the description
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_discovery_view" "tf-disco-view-desc" {
    name               = "orca-disco-view-desc"
    description        = "updated description"
    organization_level = true
    view_type          = "discovery"
    extra_params       = {}
    filter_data = {
        query = jsonencode({ "models" : ["AwsS3Bucket"], "type" : "object_set" })
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-view-desc", "description", "updated description"),
				),
			},
		},
	})
}

func TestAccDiscoveryViewResource_UpdateQuery(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_discovery_view" "tf-disco-view-1" {
    name = "orca-disco-view-1"

    organization_level = true
    view_type = "discovery"
    extra_params = {}
    filter_data = {
        query = jsonencode({
            "models": [
                "AwsS3Bucket"
            ],
            "type": "object_set"
        })
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-view-1", "filter_data.query", "jsonencode({\"models\": [\"AwsS3Bucket\"],\"type\": \"object_set\"})"),
				),
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_discovery_view" "tf-disco-view-1" {
				name = "orca-disco-view-2"

				organization_level = false
				view_type = "discovery"
				extra_params = {}
				filter_data = {
					query = jsonencode({
						"models": [
							"AzureAcrImage"
						],
						"type": "object_set"
					})
				}
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-view-1", "filter_data.query", "jsonencode({\"models\": [\"AzureAcrImage\"],\"type\": \"object_set\"})"),
				),
			},
		},
	})
}

func TestAccDiscoveryViewResource_UpdateOrgLevel(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_discovery_view" "tf-disco-view-1" {
    name = "orca-disco-view-1"

    organization_level = true
    view_type = "discovery"
    extra_params = {}
    filter_data = {
        query = jsonencode({
            "models": [
                "AwsS3Bucket"
            ],
            "type": "object_set"
        })
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-view-1", "organization_level", "true"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_discovery_view.tf-disco-view-1t",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_discovery_view" "tf-disco-view-1" {
    name = "orca-disco-view-1"

    organization_level = false
    view_type = "discovery"
    extra_params = {}
    filter_data = {
        query = jsonencode({
            "models": [
                "AwsS3Bucket"
            ],
            "type": "object_set"
        })
    }
}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-view-1", "organization_level", "false"),
				),
			},
		},
	})
}

func TestAccDiscoveryViewResource_Columns(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create with custom columns
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_discovery_view" "tf-disco-columns" {
    name = "orca-disco-columns-view"
    organization_level = false
    view_type = "discovery"
    extra_params = {}
    columns = ["OrcaScore", "CloudAccount", "AssetUniqueId"]
    sort = "-OrcaScore"
    group_by = ["AlertType"]
    filter_data = {
        query = jsonencode({
            "models": ["AwsEc2Instance"],
            "type": "object_set"
        })
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-columns", "columns.#", "3"),
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-columns", "columns.0", "OrcaScore"),
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-columns", "columns.1", "CloudAccount"),
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-columns", "columns.2", "AssetUniqueId"),
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-columns", "sort", "-OrcaScore"),
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-columns", "group_by.#", "1"),
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-columns", "group_by.0", "AlertType"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_discovery_view.tf-disco-columns",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update columns (different set and order)
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_discovery_view" "tf-disco-columns" {
    name = "orca-disco-columns-view"
    organization_level = false
    view_type = "discovery"
    extra_params = {}
    columns = ["CloudAccount", "OrcaScore", "Exposure", "Tags"]
    filter_data = {
        query = jsonencode({
            "models": ["AwsEc2Instance"],
            "type": "object_set"
        })
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-columns", "columns.#", "4"),
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-columns", "columns.0", "CloudAccount"),
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-columns", "columns.2", "Exposure"),
				),
			},
		},
	})
}

func TestAccDiscoveryViewResource_RiskFindings(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create with RiskFindings filter
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_discovery_view" "tf-disco-risk-findings" {
    name = "test-risk-findings-view"
    organization_level = false
    view_type = "discovery"
    extra_params = {}
    filter_data = {
        query = jsonencode({
            "type": "object_set",
            "models": ["Alert"],
            "with": {
                "type": "operation",
                "operator": "and",
                "values": [{
                    "key": "RiskFindings",
                    "type": "str",
                    "at_key": "code_owners.0",
                    "values": ["test-team"],
                    "operator": "containing"
                }]
            }
        })
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-risk-findings", "name", "test-risk-findings-view"),
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-risk-findings", "view_type", "discovery"),
				),
			},
			// update with RiskFindings filter
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_discovery_view" "tf-disco-risk-findings" {
    name = "test-risk-findings-view-updated"
    organization_level = false
    view_type = "discovery"
    extra_params = {}
    filter_data = {
        query = jsonencode({
            "type": "object_set",
            "models": ["Alert"],
            "with": {
                "type": "operation",
                "operator": "and",
                "values": [{
                    "key": "RiskFindings",
                    "type": "str",
                    "at_key": "code_owners.0",
                    "values": ["updated-team"],
                    "operator": "containing"
                }]
            }
        })
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_discovery_view.tf-disco-risk-findings", "name", "test-risk-findings-view-updated"),
				),
			},
		},
	})
}
