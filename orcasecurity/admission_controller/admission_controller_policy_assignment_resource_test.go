package admission_controller_test

import (
	"fmt"
	"os"
	"regexp"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccAssignmentPolicyConfig = `
data "orcasecurity_admission_controller_template" "repos_for_assignment" {
  name = "k8sallowedrepos"
}

resource "orcasecurity_admission_controller_control" "for_assignment" {
  name        = "tf-acc-assignment-control"
  template_id = data.orcasecurity_admission_controller_template.repos_for_assignment.id
  cluster_scope = {
    kinds = [
      {
        kinds      = ["Pod"]
        api_groups = [""]
        versions   = [""]
      }
    ]
  }
  input_parameters = jsonencode({ repos = ["docker.io/library"] })
}

resource "orcasecurity_admission_controller_policy" "for_assignment" {
  name     = "tf-acc-assignment-policy"
  controls = [orcasecurity_admission_controller_control.for_assignment.id]
}
`

func TestAccAdmissionControllerPolicyAssignmentResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create (full organization scope)
			{
				Config: orcasecurity.TestProviderConfig + testAccAssignmentPolicyConfig + `
resource "orcasecurity_admission_controller_policy_assignment" "test" {
  name              = "tf-acc-assignment-1"
  description       = "created"
  full_organization = true
  policy_ids        = [orcasecurity_admission_controller_policy.for_assignment.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("orcasecurity_admission_controller_policy_assignment.test", "id"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy_assignment.test", "full_organization", "true"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy_assignment.test", "policy_ids.#", "1"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy_assignment.test", "description", "created"),
				),
			},
			// ImportState
			{
				ResourceName:      "orcasecurity_admission_controller_policy_assignment.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update: rename + change description
			{
				Config: orcasecurity.TestProviderConfig + testAccAssignmentPolicyConfig + `
resource "orcasecurity_admission_controller_policy_assignment" "test" {
  name              = "tf-acc-assignment-1-renamed"
  description       = "updated"
  full_organization = true
  policy_ids        = [orcasecurity_admission_controller_policy.for_assignment.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy_assignment.test", "name", "tf-acc-assignment-1-renamed"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy_assignment.test", "description", "updated"),
				),
			},
			// Clear description and detach all policies: removing the
			// attributes must clear them remotely and converge (the client
			// sends explicit null / [] on PUT).
			{
				Config: orcasecurity.TestProviderConfig + testAccAssignmentPolicyConfig + `
resource "orcasecurity_admission_controller_policy_assignment" "test" {
  name              = "tf-acc-assignment-1-renamed"
  full_organization = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("orcasecurity_admission_controller_policy_assignment.test", "description"),
					resource.TestCheckNoResourceAttr("orcasecurity_admission_controller_policy_assignment.test", "policy_ids"),
				),
			},
		},
	})
}

// Exercises the clusters scope path (the primary real-world use case, and a
// different backend validation branch than full_organization). The backend
// validates that cluster IDs reference existing clusters, so this needs a real
// cluster ID from the target org: set ORCASECURITY_ACC_CLUSTER_ID to run it.
func TestAccAdmissionControllerPolicyAssignmentResource_Clusters(t *testing.T) {
	clusterID := os.Getenv("ORCASECURITY_ACC_CLUSTER_ID")
	if clusterID == "" {
		t.Skip("set ORCASECURITY_ACC_CLUSTER_ID to a Kubernetes cluster ID from the target org to run this test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + testAccAssignmentPolicyConfig + fmt.Sprintf(`
resource "orcasecurity_admission_controller_policy_assignment" "clusters" {
  name       = "tf-acc-assignment-clusters"
  clusters   = [%q]
  policy_ids = [orcasecurity_admission_controller_policy.for_assignment.id]
}
`, clusterID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("orcasecurity_admission_controller_policy_assignment.clusters", "id"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy_assignment.clusters", "full_organization", "false"),
					resource.TestCheckResourceAttr("orcasecurity_admission_controller_policy_assignment.clusters", "clusters.#", "1"),
					resource.TestCheckTypeSetElemAttr("orcasecurity_admission_controller_policy_assignment.clusters", "clusters.*", clusterID),
				),
			},
			{
				ResourceName:      "orcasecurity_admission_controller_policy_assignment.clusters",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAdmissionControllerPolicyAssignmentResource_RequiresScope(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_admission_controller_policy_assignment" "invalid" {
  name = "tf-acc-assignment-invalid"
}
`,
				ExpectError: regexp.MustCompile(`(?s)full_organization.*clusters.*cloud_accounts`),
			},
		},
	})
}
