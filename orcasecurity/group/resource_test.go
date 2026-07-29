package group_test

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// requireUserID returns an Orca user ID supplied through the environment. Membership cannot be
// asserted against a made-up ID: the API silently ignores users it does not know, which leaves
// the applied state without the requested member and fails the apply as an inconsistent result.
// IDs for a given org come from the /api/users endpoint (the `user_id` field).
func requireUserID(t *testing.T, envVar string) string {
	t.Helper()
	v := os.Getenv(envVar)
	if v == "" {
		t.Skipf("set %s to a user ID that exists in the target org to run this test", envVar)
	}
	return v
}

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

// TestAccGroupResource_UpdateUsers covers replacing a group's membership. It needs two user IDs
// that really exist in the target org, so it is skipped unless both are provided:
// ORCASECURITY_ACC_GROUP_USER_ID and ORCASECURITY_ACC_GROUP_USER_ID_2.
func TestAccGroupResource_UpdateUsers(t *testing.T) {
	firstUser := requireUserID(t, "ORCASECURITY_ACC_GROUP_USER_ID")
	secondUser := requireUserID(t, "ORCASECURITY_ACC_GROUP_USER_ID_2")

	config := func(userID string) string {
		return orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_group" "tf-group-1" {
    name = "Orca Terraform Group 1"

    sso_group = true
    description = "First Terraform Group"
    users = [
        %q
    ]
}
`, userID)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: config(firstUser),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_group.tf-group-1", "users.#", "1"),
					resource.TestCheckTypeSetElemAttr("orcasecurity_group.tf-group-1", "users.*", firstUser),
				),
			},
			// update — swap the member out
			{
				Config: config(secondUser),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_group.tf-group-1", "users.#", "1"),
					resource.TestCheckTypeSetElemAttr("orcasecurity_group.tf-group-1", "users.*", secondUser),
				),
			},
		},
	})
}

// TestAccGroupResource_OptionalEmptyUsers validates optional users = [] for a group with no members.
// Enable with: TF_ACC=1 ORCASECURITY_ACC_GROUP_EMPTY_USERS=1 plus ORCASECURITY_API_* credentials.
func TestAccGroupResource_OptionalEmptyUsers(t *testing.T) {
	if os.Getenv("ORCASECURITY_ACC_GROUP_EMPTY_USERS") == "" {
		t.Skip("Skipping: set ORCASECURITY_ACC_GROUP_EMPTY_USERS=1 to run this acceptance test")
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
