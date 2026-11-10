package shift_left_policy

import (
	"encoding/json"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPlanToAPI_ScmMissingScope(t *testing.T) {
	model := &shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("scm_posture"),
		Name:                     types.StringValue("scm"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		ScmPosture: &scmPostureBlockModel{
			Controls: []scmControlModel{
				{ID: types.StringValue("scm-ctrl"), Priority: types.StringValue("HIGH"), Disabled: types.BoolValue(false)},
			},
		},
	}

	_, diags := planToAPI(model)
	if !diags.HasError() {
		t.Fatal("expected error when scm_posture scope is missing")
	}
}

func TestPlanToAPI_ScmPosture(t *testing.T) {
	model := &shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("scm_posture"),
		Name:                     types.StringValue("scm"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		ScmPosture: &scmPostureBlockModel{
			Scope: []scmScopeEntryModel{
				{
					Key: types.StringValue("github_installations"),
					Ids: []types.String{types.StringValue("org-1")},
				},
			},
			Controls: []scmControlModel{
				{
					ID:       types.StringValue("scm-ctrl"),
					Priority: types.StringValue("HIGH"),
					Disabled: types.BoolValue(false),
				},
			},
		},
	}

	policy, diags := planToAPI(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(policy.Scope) == 0 {
		t.Error("expected scm scope to be encoded")
	}
}

// OOB scope keys from API must surface as drift, not be masked by prior state.

func TestAPIToState_ScmPostureScopeDriftSurfaces(t *testing.T) {
	existing := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("scm_posture"),
		ScmPosture: &scmPostureBlockModel{
			Scope: []scmScopeEntryModel{
				{Key: types.StringValue("github_installations"), Ids: []types.String{types.StringValue("org-1")}},
			},
		},
	}
	apiPolicy := &api_client.ShiftLeftPolicy{
		ID:   "policy-1",
		Type: "scm_posture",
		Scope: json.RawMessage(`{
			"github_installations": [{"id": "org-1", "name": "Org One"}],
			"gitlab_groups": [{"id": "group-1", "name": "Group One"}]
		}`),
	}

	state := apiToState(apiPolicy, existing)
	if state.ScmPosture == nil {
		t.Fatal("expected scm_posture block")
	}
	if len(state.ScmPosture.Scope) != 2 {
		t.Fatalf("expected the out-of-band gitlab_groups key to surface as drift, got %+v", state.ScmPosture.Scope)
	}
	keys := map[string]bool{}
	for _, e := range state.ScmPosture.Scope {
		keys[e.Key.ValueString()] = true
	}
	if !keys["github_installations"] || !keys["gitlab_groups"] {
		t.Fatalf("expected both scope keys present, got %+v", state.ScmPosture.Scope)
	}
}

// Sorted scope keys: reordering alone must not drift.

func TestAPIToState_ScmPostureScopeStableOrderNoDrift(t *testing.T) {
	existing := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("scm_posture"),
		ScmPosture: &scmPostureBlockModel{
			Scope: []scmScopeEntryModel{
				{Key: types.StringValue("github_installations"), Ids: []types.String{types.StringValue("org-1")}},
				{Key: types.StringValue("gitlab_groups"), Ids: []types.String{types.StringValue("group-1")}},
			},
		},
	}
	apiPolicy := &api_client.ShiftLeftPolicy{
		ID:   "policy-1",
		Type: "scm_posture",
		Scope: json.RawMessage(`{"gitlab_groups": [{"id": "group-1", "name": "Group One"}], ` +
			`"github_installations": [{"id": "org-1", "name": "Org One"}]}`),
	}

	state := apiToState(apiPolicy, existing)
	if len(state.ScmPosture.Scope) != 2 ||
		state.ScmPosture.Scope[0].Key.ValueString() != "github_installations" ||
		state.ScmPosture.Scope[1].Key.ValueString() != "gitlab_groups" {
		t.Fatalf("expected deterministic sorted scope, got %+v", state.ScmPosture.Scope)
	}
}

// Skip empty scope types; read members are {"id","name"} objects.

func TestAPIToState_ScmPostureScopeFiltersEmptyTypesAndUnwrapsMembers(t *testing.T) {
	existing := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("scm_posture"),
		ScmPosture: &scmPostureBlockModel{
			Scope: []scmScopeEntryModel{
				{Key: types.StringValue("github_installations"), Ids: []types.String{types.StringValue("org-1")}},
			},
		},
	}
	apiPolicy := &api_client.ShiftLeftPolicy{
		ID:   "policy-1",
		Type: "scm_posture",
		Scope: json.RawMessage(`{
			"github_installations": [{"id": "org-1", "name": "Org One"}],
			"gitlab_groups": [],
			"github_repository_installations": [],
			"gitlab_projects": [],
			"azure_organizations": [],
			"azure_projects": []
		}`),
	}

	state := apiToState(apiPolicy, existing)
	if len(state.ScmPosture.Scope) != 1 {
		t.Fatalf("expected empty scope types to be filtered out, got %+v", state.ScmPosture.Scope)
	}
	entry := state.ScmPosture.Scope[0]
	if entry.Key.ValueString() != "github_installations" {
		t.Fatalf("expected github_installations, got %q", entry.Key.ValueString())
	}
	if len(entry.Ids) != 1 || entry.Ids[0].ValueString() != "org-1" {
		t.Fatalf("expected id unwrapped from {id,name} member, got %+v", entry.Ids)
	}
}

func TestPlanToAPI_MaliciousPackages_NoControls(t *testing.T) {
	m := &shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("malicious_packages"),
		Name:                     types.StringValue("MP"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
	}
	policy, diags := planToAPI(m)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if policy.Type != "malicious_packages" {
		t.Errorf("type mismatch: %s", policy.Type)
	}
}
