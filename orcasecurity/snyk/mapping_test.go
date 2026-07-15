package snyk

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The shared suite covers the envelope plumbing; the closures below describe Snyk's config
// fields: the API token is write-only, and the API round-trips region.
func TestMapping(t *testing.T) {
	testutils.RunMappingSuite(t, testutils.MappingSuite[api_client.SnykConfig]{
		BuildPayload: buildPayload,
		Extract:      extract,
		TemplateName: "tf-acc-test-snyk",
		FilledState: func() cc.State {
			return &state{
				APIToken: types.StringValue("snyk-token"),
				Region:   types.StringValue("EU"),
			}
		},
		CheckConfig: func(t *testing.T, c api_client.SnykConfig) {
			if c.APIToken != "snyk-token" {
				t.Errorf("api_token mismatch: %q", c.APIToken)
			}
			if c.Region != "EU" {
				t.Errorf("region mismatch: %q", c.Region)
			}
		},
		EchoConfig: api_client.SnykConfig{Region: "US2"},
		EchoState:  func() cc.State { return &state{Region: types.StringValue("EU")} },
		CheckEchoed: func(t *testing.T, st cc.State) {
			if got := st.(*state).Region.ValueString(); got != "US2" {
				t.Errorf("Extract must echo the API region into state, got %q", got)
			}
		},
		ZeroConfigChecks: []testutils.StateCheck{
			{
				Name:  "empty region keeps planned",
				State: func() cc.State { return &state{Region: types.StringValue("AU")} },
				Check: func(t *testing.T, st cc.State) {
					if got := st.(*state).Region.ValueString(); got != "AU" {
						t.Errorf("empty API region must not clobber planned region, got %q", got)
					}
				},
			},
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
