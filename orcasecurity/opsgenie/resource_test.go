package opsgenie_test

import (
	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"
	"testing"
)

// Create/update/import via the shared simple-key runner. The backend accepts Opsgenie keys of any
// shape (verified against the lab), so a fake key works without a real Opsgenie tenant.
func TestAccOpsgenieResource(t *testing.T) {
	acctest.RunSimpleKeyIntegrationTest(t, acctest.SimpleKeyIntegrationSpec{
		ResourceType: "orcasecurity_integration_opsgenie",
		TemplateName: "tf-acc-test-opsgenie",
		KeyAttr:      "opsgenie_key",
		KeyValue:     "fake-opsgenie-key-abc123",
	})
}
