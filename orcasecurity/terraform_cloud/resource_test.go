package terraform_cloud

import (
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"
)

// api_token is the secret; api_url is a plain required URL. terraform_cloud does not support
// business_units, so the attribute must be absent.
func TestTerraformCloudResource_SchemaContract(t *testing.T) {
	testutils.CheckVariantResource(t, testutils.VariantResourceSpec{
		NewResource:   NewTerraformCloudResource,
		TypeName:      "orcasecurity_integration_terraform_cloud",
		Secrets:       []string{"api_token"},
		PlainRequired: []string{"api_url"},
		Forbidden:     []string{"business_units"},
		State:         &state{},
	})
}
