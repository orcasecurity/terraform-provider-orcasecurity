package cloudflare

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The shared suite covers the envelope plumbing; Cloudflare's config is a single write-only
// API token, so Extract never touches state.
func TestMapping(t *testing.T) {
	testutils.RunMappingSuite(t, testutils.MappingSuite[api_client.CloudflareConfig]{
		BuildPayload: buildPayload,
		Extract:      extract,
		TemplateName: "tf-acc-test-cloudflare",
		FilledState:  func() cc.State { return &state{APIToken: types.StringValue("cf-token")} },
		CheckConfig: func(t *testing.T, c api_client.CloudflareConfig) {
			if c.APIToken != "cf-token" {
				t.Errorf("api_token mismatch: %q", c.APIToken)
			}
		},
		EchoState: func() cc.State { return &state{} },
		ZeroConfigChecks: []testutils.StateCheck{
			{
				Name:  "sensitive token untouched",
				State: func() cc.State { return &state{APIToken: types.StringValue("planned-token")} },
				Check: func(t *testing.T, st cc.State) {
					if got := st.(*state).APIToken.ValueString(); got != "planned-token" {
						t.Errorf("Extract must not overwrite api_token, got %q", got)
					}
				},
			},
		},
	})
}

var _ cc.State = &state{}
