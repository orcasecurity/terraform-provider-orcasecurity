package opsgenie_test

import (
	"fmt"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	resourceType = "orcasecurity_integration_opsgenie"
	resourceName = "test"
	fullName     = resourceType + "." + resourceName
	// The backend accepts unvalidated Opsgenie keys on create (verified by probing the lab), so
	// a fake key is safe to use here without a real Opsgenie tenant.
	templateName = "tf-acc-test-opsgenie"
)

func config(key string, isDefault bool) string {
	return orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "%s" "%s" {
  template_name = "%s"
  is_default    = %t
  opsgenie_key  = "%s"
}
`, resourceType, resourceName, templateName, isDefault, key)
}

// Full lifecycle: create with a fake key, update is_default, then import. resource.Test runs
// terraform destroy at the end of the TestCase so the lab config is always torn down.
func TestAccOpsgenieResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("fake-opsgenie-key-abc123", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullName, "template_name", templateName),
					resource.TestCheckResourceAttr(fullName, "is_enabled", "true"),
					resource.TestCheckResourceAttr(fullName, "is_default", "false"),
					resource.TestCheckResourceAttr(fullName, "opsgenie_key", "fake-opsgenie-key-abc123"),
					resource.TestCheckResourceAttrWith(fullName, "id", func(v string) error {
						_, err := uuid.Parse(v)
						return err
					}),
				),
			},
			{
				Config: config("fake-opsgenie-key-abc123", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullName, "is_default", "true"),
				),
			},
			{
				ResourceName:      fullName,
				ImportState:       true,
				ImportStateId:     templateName,
				ImportStateVerify: true,
				// The API never returns the secret key, so it can't be verified on import.
				ImportStateVerifyIgnore: []string{"opsgenie_key"},
			},
		},
	})
}
