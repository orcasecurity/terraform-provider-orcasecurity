package opsgenie_test

import (
	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"
	"testing"
)

// Full lifecycle via the shared simple-key runner. The backend accepts unvalidated Opsgenie keys
// on create (verified by probing the lab), so a fake key is safe to use here without a real
// Opsgenie tenant.
func TestAccOpsgenieResource(t *testing.T) {
	acctest.RunSimpleKeyIntegrationTest(t, acctest.SimpleKeyIntegrationSpec{
		ResourceType: "orcasecurity_integration_opsgenie",
		TemplateName: "tf-acc-test-opsgenie",
		KeyAttr:      "opsgenie_key",
		KeyValue:     "fake-opsgenie-key-abc123",
	})
}
