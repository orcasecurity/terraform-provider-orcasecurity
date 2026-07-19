package pagerduty_test

import (
	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"
	"testing"
)

// Create/update/import via the shared simple-key runner. The backend validates the PagerDuty key
// format (exactly 32 characters — verified against the lab) but not its ownership, so a
// well-formed fake key works without a real PagerDuty account.
func TestAccPagerDutyResource(t *testing.T) {
	acctest.RunSimpleKeyIntegrationTest(t, acctest.SimpleKeyIntegrationSpec{
		ResourceType: "orcasecurity_integration_pagerduty",
		TemplateName: "tf-acc-test-pagerduty",
		KeyAttr:      "integration_key",
		KeyValue:     "fakeintegrationkey1234567890abcd",
	})
}
