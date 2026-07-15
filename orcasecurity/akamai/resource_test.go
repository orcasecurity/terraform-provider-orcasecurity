package akamai

import (
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"
)

// The three EdgeGrid credentials are secrets and required; host is required but not sensitive.
// Akamai uses the no-BU CommonFields flavour, so business_units must be absent.
func TestAkamaiResource_SchemaContract(t *testing.T) {
	testutils.CheckVariantResource(t, testutils.VariantResourceSpec{
		NewResource:   NewAkamaiResource,
		TypeName:      "orcasecurity_integration_akamai",
		Secrets:       []string{"access_token", "client_token", "client_secret"},
		PlainRequired: []string{"host"},
		Forbidden:     []string{"business_units"},
		State:         &state{},
	})
}
