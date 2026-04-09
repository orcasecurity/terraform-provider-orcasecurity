package group_test

import (
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

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

// TestAccGroupResource_OptionalEmptyUsers validates optional users = [] when the Orca API allows empty groups.
// Enable with: TF_ACC=1 ORCASECURITY_ACC_GROUP_EMPTY_USERS=1 plus ORCASECURITY_API_* credentials.
func TestAccGroupResource_OptionalEmptyUsers(t *testing.T) {
	if os.Getenv("ORCASECURITY_ACC_GROUP_EMPTY_USERS") == "" {
		t.Skip("Skipping: set ORCASECURITY_ACC_GROUP_EMPTY_USERS=1 to run (requires API that allows groups with no members)")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_group" "empty_users" {
  name        = "TF acc optional empty users"
  description = "Acceptance test for optional users"
  sso_group   = false
  users       = []
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_group.empty_users", "name", "TF acc optional empty users"),
					resource.TestCheckResourceAttr("orcasecurity_group.empty_users", "users.#", "0"),
				),
			},
		},
	})
}
