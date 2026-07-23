package shift_left_policy_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
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
      id       = "github_repository_missing_default_branch_protection"
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

// TestAccShiftLeftPolicyResource_BuiltinAttachProjects exercises the relaxed
// built-in guard: built-in policies cannot be Created by Terraform (Create
// would POST a brand new policy), so the only way to exercise Update's
// projects_ids-only allowance is an import-then-apply flow against a real
// built-in policy that already exists in the org.
//
// The test captures the live policy's current attached projects before
// mutating anything, only ever *adds* the scratch project (never replaces
// the existing attachment list), and restores the original list afterward.
// The final step uses a `removed` block (Terraform >= 1.7) instead of letting
// the resource fall out of config, because a real `terraform destroy` on a
// built-in policy is expected to always fail (Delete blocks it) -- forcing an
// actual destroy here would make this test permanently red for reasons
// unrelated to correctness, since terraform-plugin-testing always attempts a
// final destroy of anything left in state at the end of resource.Test.
func TestAccShiftLeftPolicyResource_BuiltinAttachProjects(t *testing.T) {
	builtinType := os.Getenv("ORCA_TEST_BUILTIN_POLICY_TYPE")
	builtinID := os.Getenv("ORCA_TEST_BUILTIN_POLICY_ID")
	projectID := os.Getenv("ORCA_TEST_PROJECT_ID")
	if builtinType == "" || builtinID == "" || projectID == "" {
		t.Skip("Set ORCA_TEST_BUILTIN_POLICY_TYPE, ORCA_TEST_BUILTIN_POLICY_ID and ORCA_TEST_PROJECT_ID to run the built-in projects-attach acceptance test")
	}
	if builtinType != "licenses" && builtinType != "sca" {
		t.Skipf("this test only knows how to render the type-specific block for licenses/sca built-ins, got %q", builtinType)
	}

	endpoint := os.Getenv("ORCASECURITY_API_ENDPOINT")
	token := os.Getenv("ORCASECURITY_API_TOKEN")
	client, err := api_client.NewAPIClient(&endpoint, &token)
	if err != nil {
		t.Fatalf("failed to build a setup API client: %s", err)
	}

	original, err := client.GetShiftLeftPolicy(builtinType, builtinID)
	if err != nil || original == nil {
		t.Fatalf("failed to read built-in policy %s/%s before test: %v", builtinType, builtinID, err)
	}

	// GetShiftLeftPolicy's ProjectsIds field never populates from the read API
	// shape (GET returns nested "projects": [{id,name,key}] objects, while the
	// write side accepts "projects_ids": [ids]), so read the currently attached
	// project ids directly from the raw body to capture a restorable baseline.
	rawResp, err := client.Get(fmt.Sprintf("/api/shiftleft/%s/policies/%s/", builtinType, builtinID))
	if err != nil {
		t.Fatalf("failed to read raw built-in policy body: %s", err)
	}
	var raw struct {
		Projects []struct {
			ID string `json:"id"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(rawResp.Body(), &raw); err != nil {
		t.Fatalf("failed to parse raw built-in policy body: %s", err)
	}
	originalProjectIDs := make([]string, 0, len(raw.Projects))
	for _, p := range raw.Projects {
		originalProjectIDs = append(originalProjectIDs, p.ID)
	}

	t.Cleanup(func() {
		restore := *original
		restore.ProjectsIds = originalProjectIDs
		if _, err := client.UpdateShiftLeftPolicy(builtinType, builtinID, restore); err != nil {
			t.Errorf("failed to restore built-in policy %s/%s project attachments after test: %s", builtinType, builtinID, err)
		}
	})

	quoteAll := func(ids []string) string {
		quoted := make([]string, len(ids))
		for i, id := range ids {
			quoted[i] = fmt.Sprintf("%q", id)
		}
		return strings.Join(quoted, ", ")
	}

	baseConfig := fmt.Sprintf(`
resource "orcasecurity_shift_left_policy" "builtin" {
  type                       = %q
  name                       = %q
  description                = %q
  disabled                   = %t
  warn_mode                  = %t
  priority_failure_threshold = %q

  %s {}
`, builtinType, original.Name, original.Description, original.Disabled, original.WarnMode, original.PriorityFailureThreshold, builtinType)

	importConfig := baseConfig + "}\n"
	attachConfig := baseConfig + fmt.Sprintf("  projects_ids = [%s]\n}\n", quoteAll(append(append([]string{}, originalProjectIDs...), projectID)))
	forgetConfig := `
removed {
  from = orcasecurity_shift_left_policy.builtin
  lifecycle {
    destroy = false
  }
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Import only: built-ins can't go through Create, so state is
				// established the same way a real user would attach projects to
				// an existing built-in -- import it, then apply projects_ids.
				Config:             orcasecurity.TestProviderConfig + importConfig,
				ResourceName:       "orcasecurity_shift_left_policy.builtin",
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateId:      builtinType + "/" + builtinID,
			},
			{
				// Only projects_ids differs from the imported state; the relaxed
				// built-in guard must allow this Update through.
				Config: orcasecurity.TestProviderConfig + attachConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.builtin", "projects_ids.#", fmt.Sprintf("%d", len(originalProjectIDs)+1)),
					resource.TestCheckTypeSetElemAttr("orcasecurity_shift_left_policy.builtin", "projects_ids.*", projectID),
				),
			},
			{
				// Forget the resource (do not destroy it -- built-in Delete is
				// intentionally always blocked; see the doc comment above).
				Config: orcasecurity.TestProviderConfig + forgetConfig,
			},
		},
	})
}

// TestAccShiftLeftPolicy_MaliciousPackages exercises the malicious_packages
// built-in policy type. Unlike licenses/sca/iac/etc, malicious_packages has no
// controls and no type-specific block at all (policy_data is always {}), so
// the config carries no `malicious_packages {}` block.
//
// Modeled on TestAccShiftLeftPolicyResource_BuiltinAttachProjects: built-ins
// cannot be Created by Terraform, so the test imports the live policy,
// attaches one scratch project via projects_ids (additively, never replacing
// the existing attachment list), verifies, then restores the original
// attachments and forgets the resource (a real destroy is expected to always
// fail for built-ins).
func TestAccShiftLeftPolicy_MaliciousPackages(t *testing.T) {
	builtinID := os.Getenv("ORCA_TEST_MALICIOUS_PACKAGES_POLICY_ID")
	if builtinID == "" {
		// Live built-in "Malicious Packages" policy id, per progress notes.
		// Re-confirm this is still current before relying on it.
		builtinID = "019efa3e-d809-797a-9b4b-eae491fc4e66"
	}
	projectID := os.Getenv("ORCA_TEST_PROJECT_ID")
	if projectID == "" {
		t.Skip("Set ORCA_TEST_PROJECT_ID to run the malicious_packages acceptance test")
	}

	endpoint := os.Getenv("ORCASECURITY_API_ENDPOINT")
	token := os.Getenv("ORCASECURITY_API_TOKEN")
	client, err := api_client.NewAPIClient(&endpoint, &token)
	if err != nil {
		t.Fatalf("failed to build a setup API client: %s", err)
	}

	original, err := client.GetShiftLeftPolicy("malicious_packages", builtinID)
	if err != nil || original == nil {
		t.Fatalf("failed to read built-in malicious_packages policy %s before test: %v", builtinID, err)
	}

	// GetShiftLeftPolicy's ProjectsIds field never populates from the read API
	// shape (GET returns nested "projects": [{id,name,key}] objects, while the
	// write side accepts "projects_ids": [ids]), so read the currently attached
	// project ids directly from the raw body to capture a restorable baseline.
	rawResp, err := client.Get(fmt.Sprintf("/api/shiftleft/malicious_packages/policies/%s/", builtinID))
	if err != nil {
		t.Fatalf("failed to read raw built-in policy body: %s", err)
	}
	var raw struct {
		Projects []struct {
			ID string `json:"id"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(rawResp.Body(), &raw); err != nil {
		t.Fatalf("failed to parse raw built-in policy body: %s", err)
	}
	originalProjectIDs := make([]string, 0, len(raw.Projects))
	for _, p := range raw.Projects {
		originalProjectIDs = append(originalProjectIDs, p.ID)
	}

	t.Cleanup(func() {
		restore := *original
		restore.ProjectsIds = originalProjectIDs
		if _, err := client.UpdateShiftLeftPolicy("malicious_packages", builtinID, restore); err != nil {
			t.Errorf("failed to restore built-in malicious_packages policy %s project attachments after test: %s", builtinID, err)
		}
	})

	quoteAll := func(ids []string) string {
		quoted := make([]string, len(ids))
		for i, id := range ids {
			quoted[i] = fmt.Sprintf("%q", id)
		}
		return strings.Join(quoted, ", ")
	}

	baseConfig := fmt.Sprintf(`
resource "orcasecurity_shift_left_policy" "malicious_packages" {
  type                       = "malicious_packages"
  name                       = %q
  description                = %q
  disabled                   = %t
  warn_mode                  = %t
  priority_failure_threshold = %q
`, original.Name, original.Description, original.Disabled, original.WarnMode, original.PriorityFailureThreshold)

	importConfig := baseConfig + "}\n"
	attachConfig := baseConfig + fmt.Sprintf("  projects_ids = [%s]\n}\n", quoteAll(append(append([]string{}, originalProjectIDs...), projectID)))
	forgetConfig := `
removed {
  from = orcasecurity_shift_left_policy.malicious_packages
  lifecycle {
    destroy = false
  }
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Import only: built-ins can't go through Create, so state is
				// established the same way a real user would attach projects to
				// an existing built-in -- import it, then apply projects_ids.
				Config:             orcasecurity.TestProviderConfig + importConfig,
				ResourceName:       "orcasecurity_shift_left_policy.malicious_packages",
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateId:      "malicious_packages/" + builtinID,
			},
			{
				// Only projects_ids differs from the imported state; the relaxed
				// built-in guard must allow this Update through.
				Config: orcasecurity.TestProviderConfig + attachConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.malicious_packages", "projects_ids.#", fmt.Sprintf("%d", len(originalProjectIDs)+1)),
					resource.TestCheckTypeSetElemAttr("orcasecurity_shift_left_policy.malicious_packages", "projects_ids.*", projectID),
				),
			},
			{
				// Forget the resource (do not destroy it -- built-in Delete is
				// intentionally always blocked; see the doc comment above).
				Config: orcasecurity.TestProviderConfig + forgetConfig,
			},
		},
	})
}
