package zscaler

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The shared suite covers the envelope plumbing; the closures below describe Zscaler's config
// fields: the OAuth secrets are write-only, and the API round-trips vanity_domain.
func TestMapping(t *testing.T) {
	testutils.RunMappingSuite(t, testutils.MappingSuite[api_client.ZscalerConfig]{
		BuildPayload: buildPayload,
		Extract:      extract,
		TemplateName: "tf-acc-test-zscaler",
		FilledState: func() cc.State {
			return &state{
				VanityDomain: types.StringValue("acme"),
				ClientID:     types.StringValue("client-id"),
				ClientSecret: types.StringValue("client-sec"),
			}
		},
		CheckConfig: func(t *testing.T, c api_client.ZscalerConfig) {
			if c.VanityDomain != "acme" {
				t.Errorf("vanity_domain mismatch: %q", c.VanityDomain)
			}
			if c.ClientID != "client-id" || c.ClientSecret != "client-sec" {
				t.Errorf("credentials mismatch: %+v", c)
			}
		},
		EchoConfig: api_client.ZscalerConfig{VanityDomain: "acme-updated"},
		EchoState:  func() cc.State { return &state{VanityDomain: types.StringValue("acme")} },
		CheckEchoed: func(t *testing.T, st cc.State) {
			if got := st.(*state).VanityDomain.ValueString(); got != "acme-updated" {
				t.Errorf("Extract must echo the API vanity_domain into state, got %q", got)
			}
		},
		ZeroConfigChecks: []testutils.StateCheck{
			{
				Name:  "empty vanity_domain keeps planned",
				State: func() cc.State { return &state{VanityDomain: types.StringValue("acme")} },
				Check: func(t *testing.T, st cc.State) {
					if got := st.(*state).VanityDomain.ValueString(); got != "acme" {
						t.Errorf("empty API vanity_domain must not clobber planned value, got %q", got)
					}
				},
			},
			{
				Name: "secrets untouched",
				State: func() cc.State {
					return &state{ClientID: types.StringValue("planned-id"), ClientSecret: types.StringValue("planned-sec")}
				},
				Check: func(t *testing.T, st cc.State) {
					s := st.(*state)
					if s.ClientID.ValueString() != "planned-id" || s.ClientSecret.ValueString() != "planned-sec" {
						t.Errorf("Extract must not overwrite secrets, got %q/%q", s.ClientID.ValueString(), s.ClientSecret.ValueString())
					}
				},
			},
		},
	})
}

var _ cc.State = &state{}
