package splunk

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The shared suite covers the envelope plumbing; the closures below describe Splunk's config
// fields: the HEC token is write-only, the API round-trips the URL, and the self-signed-cert
// flag round-trips unconditionally.
func TestMapping(t *testing.T) {
	testutils.RunMappingSuite(t, testutils.MappingSuite[api_client.SplunkConfig]{
		BuildPayload: buildPayload,
		Extract:      extract,
		TemplateName: "tf-acc-test-splunk",
		FilledState: func() cc.State {
			return &state{
				URL:                 types.StringValue("https://splunk.example.com:8088/services/collector/event"),
				Token:               types.StringValue("hec-token"),
				AllowSelfSignedCert: types.BoolValue(true),
			}
		},
		CheckConfig: func(t *testing.T, c api_client.SplunkConfig) {
			if c.URL != "https://splunk.example.com:8088/services/collector/event" {
				t.Errorf("url mismatch: %q", c.URL)
			}
			if c.Token != "hec-token" {
				t.Errorf("token mismatch: %q", c.Token)
			}
			if !c.AllowSelfSignedCert {
				t.Errorf("allow_self_signed_cert mismatch: %v", c.AllowSelfSignedCert)
			}
		},
		EchoConfig: api_client.SplunkConfig{
			URL:                 "https://returned.example.com:8088/services/collector/event",
			AllowSelfSignedCert: true,
		},
		EchoState: func() cc.State { return &state{URL: types.StringValue("https://planned")} },
		CheckEchoed: func(t *testing.T, st cc.State) {
			s := st.(*state)
			if s.URL.ValueString() != "https://returned.example.com:8088/services/collector/event" {
				t.Errorf("Extract must echo the API URL, got %q", s.URL.ValueString())
			}
			if !s.AllowSelfSignedCert.ValueBool() {
				t.Errorf("Extract must round-trip allow_self_signed_cert, got %v", s.AllowSelfSignedCert.ValueBool())
			}
		},
		ZeroConfigChecks: []testutils.StateCheck{
			{
				Name:  "empty url keeps planned",
				State: func() cc.State { return &state{URL: types.StringValue("https://planned")} },
				Check: func(t *testing.T, st cc.State) {
					if got := st.(*state).URL.ValueString(); got != "https://planned" {
						t.Errorf("empty API URL must not clobber planned URL, got %q", got)
					}
				},
			},
			{
				Name:  "sensitive token untouched",
				State: func() cc.State { return &state{Token: types.StringValue("planned-token")} },
				Check: func(t *testing.T, st cc.State) {
					if got := st.(*state).Token.ValueString(); got != "planned-token" {
						t.Errorf("Extract must not overwrite token, got %q", got)
					}
				},
			},
		},
	})
}

// The self-signed-cert flag defaults to false and must serialize as an explicit false (the API
// field has no omitempty; a missing value would be ambiguous).
func TestBuildPayload_SelfSignedCertFalse(t *testing.T) {
	s := &state{
		URL:                 types.StringValue("https://x"),
		Token:               types.StringValue("t"),
		AllowSelfSignedCert: types.BoolValue(false),
	}
	s.TemplateName = types.StringValue("tf-acc-test-splunk")

	var diags diag.Diagnostics
	got := buildPayload(context.Background(), s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.Config.AllowSelfSignedCert {
		t.Errorf("expected allow_self_signed_cert false, got true")
	}
}

var _ cc.State = &state{}
