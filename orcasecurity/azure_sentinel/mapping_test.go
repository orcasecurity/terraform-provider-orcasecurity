package azure_sentinel

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// buSet builds a types.Set of business-unit IDs matching the `business_units` attribute shape.
func buSet(t *testing.T, ids ...string) types.Set {
	t.Helper()
	vals := make([]attr.Value, 0, len(ids))
	for _, id := range ids {
		vals = append(vals, types.StringValue(id))
	}
	s, d := types.SetValue(types.StringType, vals)
	if d.HasError() {
		t.Fatalf("set build: %v", d)
	}
	return s
}

// BuildPayload must copy template/enabled/default, log_type, workspace_id, the sensitive primary
// key, and business_units into the API envelope.
func TestBuildPayload_PopulatesAllFields(t *testing.T) {
	s := &state{
		LogType:     types.StringValue("OrcaAlerts"),
		PrimaryKey:  types.StringValue("primary-secret"),
		WorkspaceID: types.StringValue("workspace-123"),
	}
	s.TemplateName = types.StringValue("tf-acc-test-sentinel")
	s.IsEnabled = types.BoolValue(true)
	s.IsDefault = types.BoolValue(false)
	s.BusinessUnits = buSet(t, "bu-1", "bu-2")

	var diags diag.Diagnostics
	got := buildPayload(context.Background(), s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.TemplateName != "tf-acc-test-sentinel" {
		t.Errorf("template_name mismatch: %q", got.TemplateName)
	}
	if got.Config.LogType != "OrcaAlerts" {
		t.Errorf("log_type mismatch: %q", got.Config.LogType)
	}
	if got.Config.WorkspaceID != "workspace-123" {
		t.Errorf("workspace_id mismatch: %q", got.Config.WorkspaceID)
	}
	if got.Config.PrimaryKey != "primary-secret" {
		t.Errorf("primary_key mismatch: %q", got.Config.PrimaryKey)
	}
	if len(got.BusinessUnits) != 2 || got.BusinessUnits[0] != "bu-1" || got.BusinessUnits[1] != "bu-2" {
		t.Errorf("business_units mismatch: %v", got.BusinessUnits)
	}
}

// A null business_units set must produce a nil slice so `omitempty` drops the field.
func TestBuildPayload_NullBusinessUnitsOmitted(t *testing.T) {
	s := &state{
		LogType:     types.StringValue("OrcaAlerts"),
		PrimaryKey:  types.StringValue("k"),
		WorkspaceID: types.StringValue("w"),
	}
	s.TemplateName = types.StringValue("tf-acc-test-sentinel")
	s.BusinessUnits = types.SetNull(types.StringType)

	var diags diag.Diagnostics
	got := buildPayload(context.Background(), s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.BusinessUnits != nil {
		t.Errorf("expected nil business_units, got %v", got.BusinessUnits)
	}
}

// Extract must map computed fields, echo log_type and workspace_id, and carry business_units
// back onto the APIObject.
func TestExtract_MapsComputedFieldsAndEchoesConfig(t *testing.T) {
	o := &api_client.AzureSentinelExternalServiceConfig{
		ID:            "uuid-sentinel",
		TemplateName:  "tf-acc-test-sentinel",
		IsEnabled:     true,
		IsDefault:     true,
		BusinessUnits: []string{"bu-1"},
		Config: api_client.AzureSentinelConfig{
			LogType:     "ReturnedLog",
			WorkspaceID: "returned-workspace",
		},
	}
	s := &state{
		LogType:     types.StringValue("OrcaAlerts"),
		WorkspaceID: types.StringValue("workspace-123"),
	}
	var diags diag.Diagnostics
	got := extract(o, s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.ID != "uuid-sentinel" || got.TemplateName != "tf-acc-test-sentinel" {
		t.Errorf("id/template mismatch: %+v", got)
	}
	if !got.IsEnabled || !got.IsDefault {
		t.Errorf("enabled/default mismatch: %+v", got)
	}
	if len(got.BusinessUnits) != 1 || got.BusinessUnits[0] != "bu-1" {
		t.Errorf("business_units mismatch: %v", got.BusinessUnits)
	}
	if s.LogType.ValueString() != "ReturnedLog" {
		t.Errorf("Extract must echo log_type, got %q", s.LogType.ValueString())
	}
	if s.WorkspaceID.ValueString() != "returned-workspace" {
		t.Errorf("Extract must echo workspace_id, got %q", s.WorkspaceID.ValueString())
	}
}

// When the API returns empty log_type / workspace_id, Extract must keep the planned values so no
// spurious diff appears.
func TestExtract_EmptyConfigKeepsPlanned(t *testing.T) {
	o := &api_client.AzureSentinelExternalServiceConfig{
		ID:           "uuid",
		TemplateName: "t",
		Config:       api_client.AzureSentinelConfig{LogType: "", WorkspaceID: ""},
	}
	s := &state{
		LogType:     types.StringValue("OrcaAlerts"),
		WorkspaceID: types.StringValue("workspace-123"),
	}
	var diags diag.Diagnostics
	extract(o, s, &diags)
	if s.LogType.ValueString() != "OrcaAlerts" {
		t.Errorf("empty API log_type must not clobber planned, got %q", s.LogType.ValueString())
	}
	if s.WorkspaceID.ValueString() != "workspace-123" {
		t.Errorf("empty API workspace_id must not clobber planned, got %q", s.WorkspaceID.ValueString())
	}
}

// Extract must not overwrite the planned sensitive primary key (API never returns it).
func TestExtract_DoesNotTouchSensitiveKey(t *testing.T) {
	o := &api_client.AzureSentinelExternalServiceConfig{ID: "uuid", TemplateName: "t"}
	s := &state{PrimaryKey: types.StringValue("planned-primary")}
	var diags diag.Diagnostics
	extract(o, s, &diags)
	if s.PrimaryKey.ValueString() != "planned-primary" {
		t.Errorf("Extract must not overwrite primary_key, got %q", s.PrimaryKey.ValueString())
	}
}

var _ cc.State = &state{}
