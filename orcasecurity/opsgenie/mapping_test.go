package opsgenie

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The shared suite covers the envelope plumbing (including business_units forwarding and null
// omission); Opsgenie's config is a single write-only API key, so Extract never touches state.
func TestMapping(t *testing.T) {
	testutils.RunMappingSuite(t, testutils.MappingSuite[api_client.OpsgenieConfig]{
		BuildPayload:          buildPayload,
		Extract:               extract,
		TemplateName:          "tf-acc-test-og",
		SupportsBusinessUnits: true,
		FilledState:           func() cc.State { return &state{OpsgenieKey: types.StringValue("secret-key")} },
		CheckConfig: func(t *testing.T, c api_client.OpsgenieConfig) {
			if c.OpsgenieKey != "secret-key" {
				t.Errorf("opsgenie_key mismatch: %q", c.OpsgenieKey)
			}
		},
		EchoState: func() cc.State { return &state{} },
		ZeroConfigChecks: []testutils.StateCheck{
			{
				Name:  "sensitive key untouched",
				State: func() cc.State { return &state{OpsgenieKey: types.StringValue("planned-secret")} },
				Check: func(t *testing.T, st cc.State) {
					if got := st.(*state).OpsgenieKey.ValueString(); got != "planned-secret" {
						t.Errorf("Extract must not overwrite the sensitive key, got %q", got)
					}
				},
			},
		},
	})
}

// compile-time guard: state must satisfy the cc.State interface used by the skeleton.
var _ cc.State = &state{}
