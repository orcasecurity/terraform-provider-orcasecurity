package group_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_group" "tf-group-1" {
    name = "Orca Terraform Group 1"
    
    sso_group = true
    description = "First Terraform Group"
    users = [
        "abc6d072-c4eb-47d3-b0c5-7c5a7ea"
    ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_group.tf-group-1", "name", "Orca Terraform Group 1"),
					resource.TestCheckResourceAttr("orcasecurity_group.tf-group-1", "sso_group", "true"),
					resource.TestCheckResourceAttr("orcasecurity_group.tf-group-1", "description", "First Terraform Group"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_group.tf-group-1",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_group" "tf-group-1" {
    name = "Orca Terraform Group 2"
    
    sso_group = false
    description = "2nd Terraform Group"
    users = [
        "abc6d072-c4eb-47d3-b0c5-7c5a7ea99g"
    ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_group.tf-group-1", "name", "Orca Terraform Group 2"),
					resource.TestCheckResourceAttr("orcasecurity_group.tf-group-1", "sso_group", "false"),
					resource.TestCheckResourceAttr("orcasecurity_group.tf-group-1", "description", "2nd Terraform Group"),
				),
			},
		},
	})
}

func TestAccGroupResource_UpdateWidgets(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_group" "tf-group-1" {
    name = "Orca Terraform Group 1"
    
    sso_group = true
    description = "First Terraform Group"
    users = [
        "abc6d072-c4eb-47d3-b0c5-7c5a7ea"
    ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_group.tf-group-1", "users[0]", "abc6d072-c4eb-47d3-b0c5-7c5a7ea"),
				),
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_group" "tf-group-1" {
    name = "Orca Terraform Group 1"
    
    sso_group = true
    description = "First Terraform Group"
    users = [
        "abc6d072-c4eb-47d3-b0c5-7c5a7ea99g"
    ]
}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_group.tf-group-1", "users[0]", "abc6d072-c4eb-47d3-b0c5-7c5a7ea99g"),
				),
			},
		},
	})
}
