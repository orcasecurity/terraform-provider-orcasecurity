package shift_left_integration

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOptionalID(t *testing.T) {
	if !OptionalID("").IsNull() {
		t.Fatal("empty id must be null")
	}
	if OptionalID("abc").ValueString() != "abc" {
		t.Fatal("non-empty id must keep value")
	}
}

func TestScmConfigFieldsFromAPI_NullProjectWhenUnbound(t *testing.T) {
	f := ScmConfigFieldsFromAPI("acme", "ENABLED", "SELECTED_REPOSITORIES", false, nil, nil, api_client.ShiftLeftConfigSettings{})
	if !f.ProjectID.IsNull() {
		t.Fatalf("expected null project_id, got %#v", f.ProjectID)
	}
	if f.IntegrationStatus.ValueString() != "ENABLED" {
		t.Fatalf("integration_status: %v", f.IntegrationStatus)
	}
	f2 := ScmConfigFieldsFromAPI("acme", "", "SELECTED_REPOSITORIES", false, nil, &api_client.ScmProjectRef{ID: "proj-1"}, api_client.ShiftLeftConfigSettings{})
	if f2.ProjectID.ValueString() != "proj-1" {
		t.Fatalf("expected project id, got %#v", f2.ProjectID)
	}
	if !f2.IntegrationStatus.IsNull() {
		t.Fatalf("empty status must be null, got %#v", f2.IntegrationStatus)
	}
}

func TestExpandConfigSettings_UnavailableAvoidScan(t *testing.T) {
	m := &ConfigSettingsModel{
		UnavailableConditions: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("AVOID_SCAN")}),
	}
	api := ExpandConfigSettings(m)
	if api.InstallationReposConfig == nil || api.InstallationReposConfig.UnavailableActions == nil {
		t.Fatalf("expected unavailable actions, got %+v", api.InstallationReposConfig)
	}
	got := api.InstallationReposConfig.UnavailableActions.Conditions
	if len(got) != 1 || got[0] != "AVOID_SCAN" {
		t.Fatalf("expected [AVOID_SCAN], got %v", got)
	}
}

func TestSharedScmConfigAttributes_HasIntegrationStatus(t *testing.T) {
	attrs := SharedScmConfigAttributes("name")
	if _, ok := attrs["integration_status"]; !ok {
		t.Fatal("expected integration_status attribute")
	}
}
