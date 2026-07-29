package shift_left_integration

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestScmConfigFieldsFromAPI_NullProjectWhenUnbound(t *testing.T) {
	f := ScmConfigFieldsFromAPI("acme", api_client.ScmUnitCommonFields{
		IntegrationStatus: "ENABLED",
		InstallationMode:  "SELECTED_REPOSITORIES",
	})
	if !f.ProjectID.IsNull() {
		t.Fatalf("expected null project_id, got %#v", f.ProjectID)
	}
	if f.IntegrationStatus.ValueString() != "ENABLED" {
		t.Fatalf("integration_status: %v", f.IntegrationStatus)
	}
	f2 := ScmConfigFieldsFromAPI("acme", api_client.ScmUnitCommonFields{
		InstallationMode: "SELECTED_REPOSITORIES",
		Project:          &api_client.ScmProjectRef{ID: "proj-1"},
	})
	if f2.ProjectID.ValueString() != "proj-1" {
		t.Fatalf("expected project id, got %#v", f2.ProjectID)
	}
	if !f2.IntegrationStatus.IsNull() {
		t.Fatalf("empty status must be null, got %#v", f2.IntegrationStatus)
	}
}

func TestScmConfigFieldsFromAPI_ReadOnlyStatusFields(t *testing.T) {
	f := ScmConfigFieldsFromAPI("acme", api_client.ScmUnitCommonFields{
		ScanAllState:                "COMPLETED",
		IntegratedRepositoriesCount: 7,
		ScmPosturePolicyID:          "sp-1",
	})
	if f.ScanAllState.ValueString() != "COMPLETED" {
		t.Fatalf("scan_all_state: %v", f.ScanAllState)
	}
	if f.IntegratedRepositoriesCount.ValueInt64() != 7 {
		t.Fatalf("integrated_repositories_count: %v", f.IntegratedRepositoriesCount)
	}
	if f.ScmPosturePolicyID.ValueString() != "sp-1" {
		t.Fatalf("scm_posture_policy_id: %v", f.ScmPosturePolicyID)
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

// The data source and the resource must report installation_mode identically, or
// the same unit reads differently depending on which you use and a data-source
// value cannot be fed into a resource config (the resource rejects raw SCAN_ALL).
func TestInstallationModeAgreesBetweenResourceAndDataSource(t *testing.T) {
	for _, mode := range []string{"SCAN_ALL", "", "SELECTED_REPOSITORIES", "SCAN_ALL_INCLUDE_FUTURE"} {
		unit := api_client.ScmUnitCommonFields{InstallationMode: mode}

		resourceValue := ScmConfigFieldsFromAPI("acme", unit).InstallationMode
		listValue, ok := SharedScmListUnitValues("acme", unit)["installation_mode"].(types.String)
		if !ok {
			t.Fatal("installation_mode must be a types.String")
		}
		if !listValue.Equal(resourceValue) {
			t.Errorf("mode %q: data source reports %v, resource reports %v", mode, listValue, resourceValue)
		}
	}
}
