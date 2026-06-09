package shift_left_policy_test

import (
	"fmt"
	"os"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const catalogControlConfig = `
data "orcasecurity_shift_left_policy_catalog_controls" "iac" {
  type = "iac"
}

locals {
  iac_control_id = [
    for c in data.orcasecurity_shift_left_policy_catalog_controls.iac.controls : c.id
    if c.id != ""
  ][0]
}

data "orcasecurity_shift_left_policy_catalog_controls" "container_image" {
  type = "container_image"
}

locals {
  container_vulnerability_control_id = one([
    for c in data.orcasecurity_shift_left_policy_catalog_controls.container_image.controls : c.id
    if c.title == "Vulnerabilities of high severity with fix available"
  ])
}
`

func TestAccShiftLeftPolicyResource_Iac(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + catalogControlConfig + `
resource "orcasecurity_shift_left_policy" "iac" {
  type                       = "iac"
  name                       = "tf-iac-policy"
  description                = "Terraform managed IaC policy"
  disabled                   = false
  warn_mode                  = false
  priority_failure_threshold = "HIGH"

  iac {
    controls {
      id       = local.iac_control_id
      priority = "HIGH"
      disabled = true
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.iac", "name", "tf-iac-policy"),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.iac", "type", "iac"),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.iac", "warn_mode", "false"),
				),
			},
			{
				ResourceName:            "orcasecurity_shift_left_policy.iac",
				ImportState:             true,
				ImportStateIdFunc:       importPolicyID("orcasecurity_shift_left_policy.iac"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"iac"},
			},
			{
				Config: orcasecurity.TestProviderConfig + catalogControlConfig + `
resource "orcasecurity_shift_left_policy" "iac" {
  type                       = "iac"
  name                       = "tf-iac-policy-updated"
  description                = "Updated description"
  disabled                   = false
  warn_mode                  = true
  priority_failure_threshold = "HIGH"

  iac {
    controls {
      id       = local.iac_control_id
      priority = "HIGH"
      disabled = true
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.iac", "name", "tf-iac-policy-updated"),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.iac", "warn_mode", "true"),
				),
			},
		},
	})
}

func TestAccShiftLeftPolicyResource_ContainerImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + catalogControlConfig + `
resource "orcasecurity_shift_left_policy" "container" {
  type                       = "container_image"
  name                       = "tf-container-policy"
  description                = "Terraform managed container image policy"
  disabled                   = false
  warn_mode                  = false
  priority_failure_threshold = "HIGH"

  container_image {
    feature_scope = ["vulnerabilities"]

    vulnerabilities {
      controls {
        id       = local.container_vulnerability_control_id
        priority = "HIGH"
        disabled = true
      }
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.container", "type", "container_image"),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.container", "name", "tf-container-policy"),
				),
			},
		},
	})
}

func TestAccShiftLeftPolicyResource_ScmPosture(t *testing.T) {
	installationID := os.Getenv("ORCASECURITY_ACC_SCM_INSTALLATION_ID")
	if installationID == "" {
		t.Skip("Set ORCASECURITY_ACC_SCM_INSTALLATION_ID to a GitHub installation ID to run SCM posture acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_policy" "scm" {
  type                       = "scm_posture"
  name                       = "tf-scm-policy"
  description                = "Terraform managed SCM posture policy"
  disabled                   = false
  warn_mode                  = false
  priority_failure_threshold = "HIGH"

  scm_posture {
    scope {
      key = "github_installations"
      ids = [%q]
    }

    controls {
      id       = "github_branch_protection_disabled"
      priority = "HIGH"
      disabled = true
    }
  }
}
`, installationID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.scm", "type", "scm_posture"),
				),
			},
		},
	})
}

func importPolicyID(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("not found: %s", resourceName)
		}
		policyType := rs.Primary.Attributes["type"]
		policyID := rs.Primary.ID
		return policyType + "/" + policyID, nil
	}
}
