package shift_left_scm_posture_default_policy

// Unit-level mapping coverage for the singleton adopt resource. The acceptance
// test (resource_test.go) is double-gated behind TF_ACC and an explicit opt-in,
// so these tests are what actually runs in ordinary CI.

import (
	"encoding/json"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestControlsToAPI_DisabledPointerSemantics(t *testing.T) {
	disabledTrue := controlModel{ID: types.StringValue("c1"), Disabled: types.BoolValue(true)}
	disabledFalse := controlModel{ID: types.StringValue("c2"), Disabled: types.BoolValue(false), Priority: types.StringValue("HIGH")}
	priorityOnly := controlModel{ID: types.StringValue("c3"), Priority: types.StringValue("LOW")}

	got := controlsToAPI([]controlModel{disabledTrue, disabledFalse, priorityOnly})
	if len(got) != 3 {
		t.Fatalf("expected 3 overrides, got %d", len(got))
	}
	if got[0].Disabled == nil || !*got[0].Disabled {
		t.Errorf("c1: expected disabled=true pointer, got %v", got[0].Disabled)
	}
	// disabled=false is an explicit override and must survive as a pointer,
	// not be collapsed into "not set".
	if got[1].Disabled == nil || *got[1].Disabled {
		t.Errorf("c2: expected disabled=false pointer, got %v", got[1].Disabled)
	}
	if got[1].Priority != "HIGH" {
		t.Errorf("c2: expected priority HIGH, got %q", got[1].Priority)
	}
	// A null disabled must stay nil so omitempty drops it from the wire.
	if got[2].Disabled != nil {
		t.Errorf("c3: expected nil disabled for unset override, got %v", *got[2].Disabled)
	}
	if got[2].Priority != "LOW" {
		t.Errorf("c3: expected priority LOW, got %q", got[2].Priority)
	}
}

func TestControlsToAPI_EmptySliceStaysEmptyNotNil(t *testing.T) {
	got := controlsToAPI([]controlModel{})
	if got == nil {
		t.Fatal("controls = [] must serialize as an empty array (clear), not null")
	}
	if len(got) != 0 {
		t.Fatalf("expected no overrides, got %v", got)
	}
}

func TestApiToState_MapsControlsAndOptionalFields(t *testing.T) {
	state, err := apiToState(&api_client.ScmPostureDefaultPolicy{
		ID:          "pol-1",
		Name:        "SCM Posture",
		Description: "built-in",
		Disabled:    true,
		PolicyData:  json.RawMessage(`{"controls":[{"id":"c1","disabled":true},{"id":"c2","priority":"CRITICAL"}]}`),
	})
	if err != nil {
		t.Fatalf("apiToState: %v", err)
	}
	if state.ID.ValueString() != "pol-1" || state.Name.ValueString() != "SCM Posture" ||
		state.Description.ValueString() != "built-in" || !state.Disabled.ValueBool() {
		t.Fatalf("top-level fields mismapped: %+v", state)
	}
	if len(state.Controls) != 2 {
		t.Fatalf("expected 2 controls, got %+v", state.Controls)
	}
	c1, c2 := state.Controls[0], state.Controls[1]
	if c1.ID.ValueString() != "c1" || !c1.Disabled.ValueBool() {
		t.Errorf("c1 mismapped: %+v", c1)
	}
	// Absent fields must map to null, not zero values, or every refresh of a
	// priority-only override would report phantom drift on disabled.
	if !c1.Priority.IsNull() {
		t.Errorf("c1: absent priority must be null, got %v", c1.Priority)
	}
	if c2.Priority.ValueString() != "CRITICAL" {
		t.Errorf("c2 mismapped: %+v", c2)
	}
	if !c2.Disabled.IsNull() {
		t.Errorf("c2: absent disabled must be null, got %v", c2.Disabled)
	}
}

func TestApiToState_NoControlsYieldsNilSlice(t *testing.T) {
	for name, raw := range map[string]json.RawMessage{
		"empty policy_data":    nil,
		"no controls key":      json.RawMessage(`{}`),
		"empty controls array": json.RawMessage(`{"controls":[]}`),
	} {
		state, err := apiToState(&api_client.ScmPostureDefaultPolicy{ID: "pol-1", PolicyData: raw})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if state.Controls != nil {
			t.Errorf("%s: expected nil Controls, got %+v", name, state.Controls)
		}
	}
}

// A malformed live policy_data must surface as an error everywhere it is
// decoded: swallowing it in the write path would PUT policy_data with empty
// controls and silently wipe every live override org-wide.
func TestDecodeFailuresPropagate(t *testing.T) {
	malformed := &api_client.ScmPostureDefaultPolicy{PolicyData: json.RawMessage(`{"controls":`)}

	if _, err := apiToState(malformed); err == nil {
		t.Error("apiToState must fail on malformed policy_data")
	}
	if _, err := liveControls(malformed); err == nil {
		t.Error("liveControls must fail on malformed policy_data")
	}
	if _, err := decodePolicyData(malformed); err == nil {
		t.Error("decodePolicyData must fail on malformed policy_data")
	}
}

// liveControls feeds the PUT body when the plan leaves controls unset, so nil
// would serialize as "controls": null instead of the explicit [] the API expects.
func TestLiveControls_AbsentControlsBecomeEmptySlice(t *testing.T) {
	got, err := liveControls(&api_client.ScmPostureDefaultPolicy{PolicyData: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("expected non-nil empty slice, got %#v", got)
	}
}
