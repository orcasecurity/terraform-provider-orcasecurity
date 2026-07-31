package shift_left_policy

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAPIToState_ProjectsIdsNullWhenUnset(t *testing.T) {
	apiPolicy := &api_client.ShiftLeftPolicy{
		ID:       "policy-1",
		Type:     "iac",
		Controls: []byte(`[{"id":"ctrl-1","priority":"HIGH","disabled":false}]`),
	}
	plan := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("iac"),
	}

	state := apiToState(apiPolicy, plan)
	if !state.ProjectsIds.IsNull() {
		t.Errorf("expected null projects_ids, got %#v", state.ProjectsIds)
	}
}

func TestAPIToState_ProjectsIdsPopulatedFromInstance(t *testing.T) {
	apiPolicy := &api_client.ShiftLeftPolicy{
		ID:          "policy-1",
		Type:        "licenses",
		Builtin:     true,
		ProjectsIds: []string{"a", "b"},
		PolicyData:  []byte(`{"controls":[]}`),
	}

	state := apiToState(apiPolicy, nil)
	got := tfconv.SetToStringSlice(state.ProjectsIds)
	if len(got) != 2 {
		t.Fatalf("expected 2 projects_ids, got %#v", got)
	}
	if got[0] != "a" || got[1] != "b" {
		t.Errorf("expected [a b], got %#v", got)
	}
}

// Read must reflect API projects_ids even when prior state had none.

func TestAPIToState_ProjectsIdsAuthoritativeOnRead(t *testing.T) {
	existing := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("licenses"), ProjectsIds: types.SetNull(types.StringType),
	}
	api := &api_client.ShiftLeftPolicy{
		ID: "p1", Type: "licenses", ProjectsIds: []string{"proj-a", "proj-b"},
	}
	state := apiToState(api, existing)
	got := tfconv.SetToStringSlice(state.ProjectsIds)
	if len(got) != 2 {
		t.Fatalf("expected refresh to reflect API projects [proj-a proj-b], got %v", got)
	}
}

func TestAPIToState_ProjectsIdsEmptyStaysNull(t *testing.T) {
	existing := &shiftLeftPolicyResourceModel{Type: types.StringValue("licenses")}
	api := &api_client.ShiftLeftPolicy{ID: "p1", Type: "licenses"}
	state := apiToState(api, existing)
	if !state.ProjectsIds.IsNull() {
		t.Fatalf("expected null, got %v", state.ProjectsIds)
	}
}

func TestAPIToState_ProjectsIdsEmptySetSurvivesRefresh(t *testing.T) {
	existing := &shiftLeftPolicyResourceModel{
		Type:        types.StringValue("licenses"),
		ProjectsIds: types.SetValueMust(types.StringType, nil),
	}
	api := &api_client.ShiftLeftPolicy{ID: "p1", Type: "licenses"}
	state := apiToState(api, existing)
	if state.ProjectsIds.IsNull() {
		t.Fatal("an explicitly empty projects_ids must stay [] after refresh, not become null")
	}
	if got := tfconv.SetToStringSlice(state.ProjectsIds); len(got) != 0 {
		t.Fatalf("expected an empty set, got %v", got)
	}
}

func TestAllControlsScopeKeys(t *testing.T) {
	topLevel := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("iac"),
		Iac:  &iacBlockModel{AllControls: types.BoolValue(true)},
	}
	keys := allControlsScopeKeys(topLevel)
	if len(keys) != 1 || keys[0] != "" {
		t.Errorf("expected top-level all_controls to map to [\"\"], got %+v", keys)
	}

	notRequested := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("iac"),
		Iac:  &iacBlockModel{AllControls: types.BoolValue(false)},
	}
	if keys := allControlsScopeKeys(notRequested); keys != nil {
		t.Errorf("expected nil when all_controls is false, got %+v", keys)
	}

	container := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("container_image"),
		ContainerImage: &containerImageBlockModel{
			Vulnerabilities: &containerScopeBlockModel{AllControls: types.BoolValue(true)},
			SecretDetection: &containerScopeBlockModel{AllControls: types.BoolValue(false)},
		},
	}
	keys = allControlsScopeKeys(container)
	if len(keys) != 1 || keys[0] != "vulnerabilities" {
		t.Errorf("expected [vulnerabilities], got %+v", keys)
	}
}

// Covers file_system_* single-scope handlers (not exercised by TestAllControlsScopeKeys).

func TestAllControlsScopeKeys_FsScopedHandler(t *testing.T) {
	vulnRequested := &shiftLeftPolicyResourceModel{
		Type:                      types.StringValue("file_system_vulnerabilities"),
		FileSystemVulnerabilities: &controlsBlockModel{AllControls: types.BoolValue(true)},
	}
	if keys := allControlsScopeKeys(vulnRequested); len(keys) != 1 || keys[0] != "vulnerabilities" {
		t.Errorf("expected [vulnerabilities], got %+v", keys)
	}

	vulnNotRequested := &shiftLeftPolicyResourceModel{
		Type:                      types.StringValue("file_system_vulnerabilities"),
		FileSystemVulnerabilities: &controlsBlockModel{AllControls: types.BoolValue(false)},
	}
	if keys := allControlsScopeKeys(vulnNotRequested); keys != nil {
		t.Errorf("expected nil when all_controls is false, got %+v", keys)
	}

	secretRequested := &shiftLeftPolicyResourceModel{
		Type:                      types.StringValue("file_system_secret_detection"),
		FileSystemSecretDetection: &controlsBlockModel{AllControls: types.BoolValue(true)},
	}
	if keys := allControlsScopeKeys(secretRequested); len(keys) != 1 || keys[0] != "secret_detection" {
		t.Errorf("expected [secret_detection], got %+v", keys)
	}

	// file_system_* types must not read the sibling scope block.
	secretNilBlock := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("file_system_secret_detection"),
		FileSystemVulnerabilities: &controlsBlockModel{
			AllControls: types.BoolValue(true),
		},
		FileSystemSecretDetection: nil,
	}
	if keys := allControlsScopeKeys(secretNilBlock); keys != nil {
		t.Errorf("expected nil when the type's own block is unset, got %+v", keys)
	}
}
