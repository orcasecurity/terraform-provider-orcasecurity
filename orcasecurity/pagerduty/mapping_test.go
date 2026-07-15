package pagerduty

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The shared suite covers the envelope plumbing; PagerDuty's config is a single write-only
// integration key, so Extract never touches state.
func TestMapping(t *testing.T) {
	testutils.RunMappingSuite(t, testutils.MappingSuite[api_client.PagerDutyConfig]{
		BuildPayload: buildPayload,
		Extract:      extract,
		TemplateName: "tf-acc-test-pd",
		FilledState:  func() cc.State { return &state{IntegrationKey: types.StringValue("int-key")} },
		CheckConfig: func(t *testing.T, c api_client.PagerDutyConfig) {
			if c.IntegrationKey != "int-key" {
				t.Errorf("integration_key mismatch: %q", c.IntegrationKey)
			}
		},
		EchoState: func() cc.State { return &state{} },
		ZeroConfigChecks: []testutils.StateCheck{
			{
				Name:  "sensitive key untouched",
				State: func() cc.State { return &state{IntegrationKey: types.StringValue("planned-key")} },
				Check: func(t *testing.T, st cc.State) {
					if got := st.(*state).IntegrationKey.ValueString(); got != "planned-key" {
						t.Errorf("Extract must not overwrite integration_key, got %q", got)
					}
				},
			},
		},
	})
}

// An empty integration key must serialize to an empty Config.IntegrationKey so the update path's
// `omitempty` drops it and the API keeps the secret already in SSM.
func TestBuildPayload_EmptyKeyProducesEmptyString(t *testing.T) {
	s := &state{IntegrationKey: types.StringNull()}
	s.TemplateName = types.StringValue("tf-acc-test-pd")

	var diags diag.Diagnostics
	got := buildPayload(context.Background(), s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.Config.IntegrationKey != "" {
		t.Errorf("expected empty integration_key, got %q", got.Config.IntegrationKey)
	}
}

var _ cc.State = &state{}
