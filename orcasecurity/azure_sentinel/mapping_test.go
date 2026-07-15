package azure_sentinel

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The shared suite covers the envelope plumbing (including business_units forwarding and null
// omission); the closures below describe Azure Sentinel's config fields: the primary key is
// write-only, and the API round-trips log_type and workspace_id.
func TestMapping(t *testing.T) {
	testutils.RunMappingSuite(t, testutils.MappingSuite[api_client.AzureSentinelConfig]{
		BuildPayload:          buildPayload,
		Extract:               extract,
		TemplateName:          "tf-acc-test-sentinel",
		SupportsBusinessUnits: true,
		FilledState: func() cc.State {
			return &state{
				LogType:     types.StringValue("OrcaAlerts"),
				PrimaryKey:  types.StringValue("primary-secret"),
				WorkspaceID: types.StringValue("workspace-123"),
			}
		},
		CheckConfig: func(t *testing.T, c api_client.AzureSentinelConfig) {
			testutils.AssertEq(t, "log_type", c.LogType, "OrcaAlerts")
			testutils.AssertEq(t, "workspace_id", c.WorkspaceID, "workspace-123")
			testutils.AssertEq(t, "primary_key", c.PrimaryKey, "primary-secret")
		},
		EchoConfig: api_client.AzureSentinelConfig{LogType: "ReturnedLog", WorkspaceID: "returned-workspace"},
		EchoState: func() cc.State {
			return &state{
				LogType:     types.StringValue("OrcaAlerts"),
				WorkspaceID: types.StringValue("workspace-123"),
			}
		},
		CheckEchoed: func(t *testing.T, st cc.State) {
			s := st.(*state)
			testutils.AssertEq(t, "echoed log_type", s.LogType.ValueString(), "ReturnedLog")
			testutils.AssertEq(t, "echoed workspace_id", s.WorkspaceID.ValueString(), "returned-workspace")
		},
		ZeroConfigChecks: []testutils.StateCheck{
			{
				Name: "empty config keeps planned",
				State: func() cc.State {
					return &state{
						LogType:     types.StringValue("OrcaAlerts"),
						WorkspaceID: types.StringValue("workspace-123"),
					}
				},
				Check: func(t *testing.T, st cc.State) {
					s := st.(*state)
					testutils.AssertEq(t, "planned log_type", s.LogType.ValueString(), "OrcaAlerts")
					testutils.AssertEq(t, "planned workspace_id", s.WorkspaceID.ValueString(), "workspace-123")
				},
			},
			{
				Name:  "sensitive primary key untouched",
				State: func() cc.State { return &state{PrimaryKey: types.StringValue("planned-primary")} },
				Check: func(t *testing.T, st cc.State) {
					testutils.AssertEq(t, "planned primary_key", st.(*state).PrimaryKey.ValueString(), "planned-primary")
				},
			},
		},
	})
}

var _ cc.State = &state{}
