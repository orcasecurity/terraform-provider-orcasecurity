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
			if c.LogType != "OrcaAlerts" {
				t.Errorf("log_type mismatch: %q", c.LogType)
			}
			if c.WorkspaceID != "workspace-123" {
				t.Errorf("workspace_id mismatch: %q", c.WorkspaceID)
			}
			if c.PrimaryKey != "primary-secret" {
				t.Errorf("primary_key mismatch: %q", c.PrimaryKey)
			}
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
			if s.LogType.ValueString() != "ReturnedLog" {
				t.Errorf("Extract must echo log_type, got %q", s.LogType.ValueString())
			}
			if s.WorkspaceID.ValueString() != "returned-workspace" {
				t.Errorf("Extract must echo workspace_id, got %q", s.WorkspaceID.ValueString())
			}
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
					if s.LogType.ValueString() != "OrcaAlerts" {
						t.Errorf("empty API log_type must not clobber planned, got %q", s.LogType.ValueString())
					}
					if s.WorkspaceID.ValueString() != "workspace-123" {
						t.Errorf("empty API workspace_id must not clobber planned, got %q", s.WorkspaceID.ValueString())
					}
				},
			},
			{
				Name:  "sensitive primary key untouched",
				State: func() cc.State { return &state{PrimaryKey: types.StringValue("planned-primary")} },
				Check: func(t *testing.T, st cc.State) {
					if got := st.(*state).PrimaryKey.ValueString(); got != "planned-primary" {
						t.Errorf("Extract must not overwrite primary_key, got %q", got)
					}
				},
			},
		},
	})
}

var _ cc.State = &state{}
