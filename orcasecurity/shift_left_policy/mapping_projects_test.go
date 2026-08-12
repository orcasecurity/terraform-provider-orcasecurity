package shift_left_policy

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	got := tfconv.ListToStringSlice(state.ProjectsIds)
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
		Type: types.StringValue("licenses"), ProjectsIds: types.ListNull(types.StringType),
	}
	api := &api_client.ShiftLeftPolicy{
		ID: "p1", Type: "licenses", ProjectsIds: []string{"proj-a", "proj-b"},
	}
	state := apiToState(api, existing)
	got := tfconv.ListToStringSlice(state.ProjectsIds)
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
		ProjectsIds: types.ListValueMust(types.StringType, nil),
	}
	api := &api_client.ShiftLeftPolicy{ID: "p1", Type: "licenses"}
	state := apiToState(api, existing)
	if state.ProjectsIds.IsNull() {
		t.Fatal("an explicitly empty projects_ids must stay [] after refresh, not become null")
	}
	if got := tfconv.ListToStringSlice(state.ProjectsIds); len(got) != 0 {
		t.Fatalf("expected an empty list, got %v", got)
	}
}

// The API doesn't guarantee a stable order for projects_ids. Read must reorder its response to
// match the prior state's order so an unchanged attached set doesn't drift on every refresh.
func TestAPIToState_ProjectsIdsReorderedToMatchPriorOnUnstableAPIOrder(t *testing.T) {
	existing := &shiftLeftPolicyResourceModel{
		Type:        types.StringValue("licenses"),
		ProjectsIds: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("proj-a"), types.StringValue("proj-b")}),
	}
	api := &api_client.ShiftLeftPolicy{
		ID: "p1", Type: "licenses", ProjectsIds: []string{"proj-b", "proj-a"},
	}
	state := apiToState(api, existing)
	if !state.ProjectsIds.Equal(existing.ProjectsIds) {
		t.Fatalf("expected reorder to match prior state order [proj-a proj-b], got %v", state.ProjectsIds)
	}
}

// A genuine membership change (project added) must still surface as a real diff, with the new
// entry appended after the elements that stayed in their prior order.
func TestAPIToState_ProjectsIdsNewMembersAppendedAfterReorder(t *testing.T) {
	existing := &shiftLeftPolicyResourceModel{
		Type:        types.StringValue("licenses"),
		ProjectsIds: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("proj-a"), types.StringValue("proj-b")}),
	}
	api := &api_client.ShiftLeftPolicy{
		ID: "p1", Type: "licenses", ProjectsIds: []string{"proj-c", "proj-b", "proj-a"},
	}
	state := apiToState(api, existing)
	got := tfconv.ListToStringSlice(state.ProjectsIds)
	want := []string{"proj-a", "proj-b", "proj-c"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
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
