package cloudflare

import (
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"
)

// api_token is the only credential and must be Required + Sensitive; cloudflare does not support
// business_units, so the attribute must be absent.
func TestCloudflareResource_SchemaContract(t *testing.T) {
	testutils.CheckVariantResource(t, testutils.VariantResourceSpec{
		NewResource: NewCloudflareResource,
		TypeName:    "orcasecurity_integration_cloudflare",
		Secrets:     []string{"api_token"},
		Forbidden:   []string{"business_units"},
		State:       &state{},
	})
}
