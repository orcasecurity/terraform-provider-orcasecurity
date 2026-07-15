package zscaler

import (
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"
)

// The OAuth client_id/client_secret are secrets; vanity_domain is a plain identifier. zscaler
// does not support business_units, so the attribute must be absent.
func TestZscalerResource_SchemaContract(t *testing.T) {
	testutils.CheckVariantResource(t, testutils.VariantResourceSpec{
		NewResource:   NewZscalerResource,
		TypeName:      "orcasecurity_integration_zscaler_zpa",
		Secrets:       []string{"client_id", "client_secret"},
		PlainRequired: []string{"vanity_domain"},
		Forbidden:     []string{"business_units"},
		State:         &state{},
	})
}
