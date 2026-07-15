package pagerduty_test

import (
	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"
	"testing"
)

// Full lifecycle via the shared simple-key runner. The backend accepts unvalidated PagerDuty
// integration keys on create (verified by probing the lab), so a fake key is safe here without a
// real PagerDuty account.
func TestAccPagerDutyResource(t *testing.T) {
	acctest.RunSimpleKeyIntegrationTest(t, acctest.SimpleKeyIntegrationSpec{
		ResourceType: "orcasecurity_integration_pagerduty",
		TemplateName: "tf-acc-test-pagerduty",
		KeyAttr:      "integration_key",
		KeyValue:     "fakeintegrationkey1234567890abcd",
	})
}
