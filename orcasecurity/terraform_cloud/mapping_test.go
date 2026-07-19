package terraform_cloud

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The shared suite covers the envelope plumbing; the closures below describe Terraform Cloud's
// config fields: the API token is write-only, and the API round-trips api_url.
func TestMapping(t *testing.T) {
	testutils.RunMappingSuite(t, testutils.MappingSuite[api_client.TerraformCloudConfig]{
		BuildPayload: buildPayload,
		Extract:      extract,
		TemplateName: "tf-acc-test-tfc",
		FilledState: func() cc.State {
			return &state{
				APIToken: types.StringValue("tfc-token"),
				APIURL:   types.StringValue("https://app.terraform.io"),
			}
		},
		CheckConfig: func(t *testing.T, c api_client.TerraformCloudConfig) {
			if c.APIToken != "tfc-token" {
				t.Errorf("api_token mismatch: %q", c.APIToken)
			}
			if c.APIURL != "https://app.terraform.io" {
				t.Errorf("api_url mismatch: %q", c.APIURL)
			}
		},
		EchoConfig: api_client.TerraformCloudConfig{APIURL: "https://tfe.example.com"},
		EchoState:  func() cc.State { return &state{APIURL: types.StringValue("https://app.terraform.io")} },
		CheckEchoed: func(t *testing.T, st cc.State) {
			if got := st.(*state).APIURL.ValueString(); got != "https://tfe.example.com" {
				t.Errorf("Extract must echo the API api_url into state, got %q", got)
			}
		},
		ZeroConfigChecks: []testutils.StateCheck{
			{
				Name:  "empty api_url keeps planned",
				State: func() cc.State { return &state{APIURL: types.StringValue("https://app.terraform.io")} },
				Check: func(t *testing.T, st cc.State) {
					if got := st.(*state).APIURL.ValueString(); got != "https://app.terraform.io" {
						t.Errorf("empty API api_url must not clobber planned value, got %q", got)
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
