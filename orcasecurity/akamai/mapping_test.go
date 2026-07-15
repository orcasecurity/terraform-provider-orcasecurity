package akamai

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The shared suite covers the envelope plumbing; the closures below describe Akamai's config
// fields: the three EdgeGrid secrets are write-only, and the API round-trips host.
func TestMapping(t *testing.T) {
	testutils.RunMappingSuite(t, testutils.MappingSuite[api_client.AkamaiConfig]{
		BuildPayload: buildPayload,
		Extract:      extract,
		TemplateName: "tf-acc-test-akamai",
		FilledState: func() cc.State {
			return &state{
				AccessToken:  types.StringValue("acc-tok"),
				ClientToken:  types.StringValue("cli-tok"),
				ClientSecret: types.StringValue("cli-sec"),
				Host:         types.StringValue("akab-x.luna.akamaiapis.net"),
			}
		},
		CheckConfig: func(t *testing.T, c api_client.AkamaiConfig) {
			if c.AccessToken != "acc-tok" || c.ClientToken != "cli-tok" || c.ClientSecret != "cli-sec" {
				t.Errorf("credentials mismatch: %+v", c)
			}
			if c.Host != "akab-x.luna.akamaiapis.net" {
				t.Errorf("host mismatch: %q", c.Host)
			}
		},
		EchoConfig: api_client.AkamaiConfig{Host: "akab-y.luna.akamaiapis.net"},
		EchoState:  func() cc.State { return &state{Host: types.StringValue("akab-x.luna.akamaiapis.net")} },
		CheckEchoed: func(t *testing.T, st cc.State) {
			if got := st.(*state).Host.ValueString(); got != "akab-y.luna.akamaiapis.net" {
				t.Errorf("Extract must echo the API host into state, got %q", got)
			}
		},
		ZeroConfigChecks: []testutils.StateCheck{
			{
				Name:  "empty host keeps planned",
				State: func() cc.State { return &state{Host: types.StringValue("akab-x.luna.akamaiapis.net")} },
				Check: func(t *testing.T, st cc.State) {
					if got := st.(*state).Host.ValueString(); got != "akab-x.luna.akamaiapis.net" {
						t.Errorf("empty API host must not clobber planned host, got %q", got)
					}
				},
			},
			{
				Name: "secrets untouched",
				State: func() cc.State {
					return &state{
						AccessToken:  types.StringValue("planned-acc"),
						ClientToken:  types.StringValue("planned-cli"),
						ClientSecret: types.StringValue("planned-sec"),
					}
				},
				Check: func(t *testing.T, st cc.State) {
					s := st.(*state)
					if s.AccessToken.ValueString() != "planned-acc" || s.ClientToken.ValueString() != "planned-cli" || s.ClientSecret.ValueString() != "planned-sec" {
						t.Errorf("Extract must not overwrite secrets, got %q/%q/%q",
							s.AccessToken.ValueString(), s.ClientToken.ValueString(), s.ClientSecret.ValueString())
					}
				},
			},
		},
	})
}

var _ cc.State = &state{}
