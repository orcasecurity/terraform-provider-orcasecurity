// Package acctest holds acceptance-test runners shared by external _test packages. It is imported
// exclusively from _test.go files, so nothing here ships in the provider binary.
package acctest

import (
	"fmt"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// SimpleKeyIntegrationSpec describes a config-integration variant whose only variant attribute is
// a single sensitive key the API never returns (opsgenie, pagerduty).
type SimpleKeyIntegrationSpec struct {
	// ResourceType is the fully qualified resource type, e.g. orcasecurity_integration_opsgenie.
	ResourceType string
	// TemplateName is the config name used both in the HCL and as the import ID.
	TemplateName string
	// KeyAttr is the HCL name of the single sensitive key attribute.
	KeyAttr string
	// KeyValue is the fake key to create with. It must satisfy any backend-side format checks
	// (e.g. PagerDuty enforces a 32-character key) but does not need to belong to a real account.
	KeyValue string
}

// RunSimpleKeyIntegrationTest exercises create (with defaults asserted), an in-place update that
// flips both is_default and is_enabled, and import. It does not cover business_units or the
// template_name RequiresReplace path. resource.Test runs terraform destroy at the end of the
// TestCase so the lab config is always torn down.
func RunSimpleKeyIntegrationTest(t *testing.T, spec SimpleKeyIntegrationSpec) {
	fullName := spec.ResourceType + ".test"
	config := func(isDefault, isEnabled bool) string {
		return orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "%s" "test" {
  template_name = "%s"
  is_default    = %t
  is_enabled    = %t
  %s = "%s"
}
`, spec.ResourceType, spec.TemplateName, isDefault, isEnabled, spec.KeyAttr, spec.KeyValue)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(false, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullName, "template_name", spec.TemplateName),
					resource.TestCheckResourceAttr(fullName, "is_enabled", "true"),
					resource.TestCheckResourceAttr(fullName, "is_default", "false"),
					resource.TestCheckResourceAttr(fullName, spec.KeyAttr, spec.KeyValue),
					resource.TestCheckResourceAttrWith(fullName, "id", func(v string) error {
						_, err := uuid.Parse(v)
						return err
					}),
				),
			},
			{
				Config: config(true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullName, "is_default", "true"),
					resource.TestCheckResourceAttr(fullName, "is_enabled", "false"),
				),
			},
			{
				ResourceName:      fullName,
				ImportState:       true,
				ImportStateId:     spec.TemplateName,
				ImportStateVerify: true,
				// The API never returns the secret key, so it can't be verified on import.
				ImportStateVerifyIgnore: []string{spec.KeyAttr},
			},
		},
	})
}
