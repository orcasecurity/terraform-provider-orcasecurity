package shift_left_policy

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func TestPlanToAPI_Iac(t *testing.T) {
	model := &shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("iac"),
		Name:                     types.StringValue("IaC baseline"),
		Description:              types.StringValue("desc"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		Iac: &iacBlockModel{
			Controls: []iacControlModel{
				{
					baseControlModel: baseControlModel{
						ID:       types.StringValue("ctrl-1"),
						Priority: types.StringValue("HIGH"),
						Disabled: types.BoolValue(false),
					},
				},
			},
		},
	}

	policy, diags := planToAPI(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if policy.Name != "IaC baseline" {
		t.Errorf("expected name IaC baseline, got %s", policy.Name)
	}
	if len(policy.Controls) == 0 || len(policy.PolicyData) == 0 {
		t.Error("expected controls and policy_data to be set")
	}
}

func TestPlanToAPI_MissingBlock(t *testing.T) {
	model := &shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("iac"),
		Name:                     types.StringValue("test"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
	}
	_, diags := planToAPI(model)
	if !diags.HasError() {
		t.Fatal("expected error for missing iac block")
	}
}

func TestAPIToState_Iac(t *testing.T) {
	apiPolicy := &api_client.ShiftLeftPolicy{
		ID:                       "policy-1",
		Type:                     "iac",
		Name:                     "IaC baseline",
		Description:              "desc",
		Disabled:                 false,
		WarnMode:                 false,
		PriorityFailureThreshold: "HIGH",
		Controls:                 []byte(`[{"id":"ctrl-1","priority":"HIGH","disabled":false}]`),
		PolicyData:               []byte(`{"controls":[{"id":"ctrl-1","priority":"HIGH","disabled":false}]}`),
	}

	state := apiToState(apiPolicy, nil)
	if state.Iac == nil || len(state.Iac.Controls) != 1 {
		t.Fatalf("expected one iac control, got %+v", state.Iac)
	}
	if state.Iac.Controls[0].ID.ValueString() != "ctrl-1" {
		t.Errorf("expected ctrl-1, got %s", state.Iac.Controls[0].ID.ValueString())
	}
}

func TestAPIToState_ContainerImageResolvesControlID(t *testing.T) {
	apiPolicy := &api_client.ShiftLeftPolicy{
		ID:   "policy-1",
		Type: "container_image",
		Controls: []byte(`[{"id":"vuln-1","priority":"HIGH","disabled":true,"title":"Vulnerabilities of high severity with fix available"}]`),
		PolicyData: []byte(`{
			"feature_scope":["vulnerabilities"],
			"vulnerabilities":{"controls":[{"priority":"HIGH","disabled":true,"title":"Vulnerabilities of high severity with fix available","conditions":{"fix_available":true}}]}
		}`),
		FeatureScope: []string{"vulnerabilities"},
	}
	plan := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("container_image"),
		ContainerImage: &containerImageBlockModel{
			Vulnerabilities: &containerScopeBlockModel{
				Controls: []containerControlModel{
					{
						baseControlModel: baseControlModel{
							ID:       types.StringValue("vuln-1"),
							Priority: types.StringValue("HIGH"),
							Disabled: types.BoolValue(true),
						},
					},
				},
			},
		},
	}

	state := apiToState(apiPolicy, plan)
	if state.ContainerImage == nil || state.ContainerImage.Vulnerabilities == nil {
		t.Fatal("expected container vulnerabilities block")
	}
	if len(state.ContainerImage.Vulnerabilities.Controls) != 1 {
		t.Fatalf("expected one control, got %d", len(state.ContainerImage.Vulnerabilities.Controls))
	}
	ctrl := state.ContainerImage.Vulnerabilities.Controls[0]
	if ctrl.ID.ValueString() != "vuln-1" {
		t.Errorf("expected vuln-1, got %s", ctrl.ID.ValueString())
	}
	if !ctrl.Title.IsNull() {
		t.Errorf("expected title to be omitted when not configured in plan, got %s", ctrl.Title.ValueString())
	}
	if ctrl.Conditions != nil {
		t.Error("expected conditions to be omitted when not configured in plan")
	}
}

func TestStateFromPlanAfterWrite(t *testing.T) {
	plan := &shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("container_image"),
		Name:                     types.StringValue("tf-container-policy"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		ContainerImage: &containerImageBlockModel{
			FeatureScope: []types.String{types.StringValue("vulnerabilities")},
			Vulnerabilities: &containerScopeBlockModel{
				Controls: []containerControlModel{
					{
						baseControlModel: baseControlModel{
							ID:       types.StringValue("vuln-1"),
							Priority: types.StringValue("HIGH"),
							Disabled: types.BoolValue(true),
						},
					},
				},
			},
		},
	}
	apiPolicy := &api_client.ShiftLeftPolicy{
		ID:      "policy-1",
		Builtin: false,
	}

	state := stateFromPlanAfterWrite(plan, apiPolicy)
	if state.ID.ValueString() != "policy-1" {
		t.Fatalf("expected policy-1, got %s", state.ID.ValueString())
	}
	if state.ContainerImage.Vulnerabilities.Controls[0].ID.ValueString() != "vuln-1" {
		t.Fatalf("expected plan control id to be preserved, got %s", state.ContainerImage.Vulnerabilities.Controls[0].ID.ValueString())
	}
}

func TestAPIToState_ContainerImagePrefersPlanControlID(t *testing.T) {
	apiPolicy := &api_client.ShiftLeftPolicy{
		ID:   "policy-1",
		Type: "container_image",
		Controls: []byte(`[
			{"id":"wrong-id","priority":"HIGH","disabled":true,"title":"Other control"},
			{"id":"vuln-1","priority":"HIGH","disabled":true,"title":"Vulnerabilities of high severity with fix available"}
		]`),
		PolicyData: []byte(`{
			"feature_scope":["vulnerabilities"],
			"vulnerabilities":{"controls":[{"priority":"HIGH","disabled":true,"title":"Vulnerabilities of high severity with fix available"}]}
		}`),
		FeatureScope: []string{"vulnerabilities"},
	}
	plan := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("container_image"),
		ContainerImage: &containerImageBlockModel{
			Vulnerabilities: &containerScopeBlockModel{
				Controls: []containerControlModel{
					{
						baseControlModel: baseControlModel{
							ID:       types.StringValue("vuln-1"),
							Priority: types.StringValue("HIGH"),
							Disabled: types.BoolValue(true),
						},
					},
				},
			},
		},
	}

	state := apiToState(apiPolicy, plan)
	ctrl := state.ContainerImage.Vulnerabilities.Controls[0]
	if ctrl.ID.ValueString() != "vuln-1" {
		t.Errorf("expected plan control id vuln-1, got %s", ctrl.ID.ValueString())
	}
}

func TestAPIToState_ProjectsIdsNullWhenUnset(t *testing.T) {
	apiPolicy := &api_client.ShiftLeftPolicy{
		ID:   "policy-1",
		Type: "iac",
		Controls: []byte(`[{"id":"ctrl-1","priority":"HIGH","disabled":false}]`),
	}
	plan := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("iac"),
	}

	state := apiToState(apiPolicy, plan)
	if state.ProjectsIds != nil {
		t.Errorf("expected nil projects_ids, got %#v", state.ProjectsIds)
	}
}

func TestParseImportID(t *testing.T) {
	policyType, policyID, err := parseImportID("iac/abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policyType != "iac" || policyID != "abc-123" {
		t.Errorf("got %s/%s", policyType, policyID)
	}

	_, _, err = parseImportID("invalid")
	if err == nil {
		t.Fatal("expected error for invalid import id")
	}
}

func TestPlanToAPI_ContainerImage(t *testing.T) {
	model := &shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("container_image"),
		Name:                     types.StringValue("image policy"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		ContainerImage: &containerImageBlockModel{
			FeatureScope: []types.String{types.StringValue("vulnerabilities")},
			Vulnerabilities: &containerScopeBlockModel{
				Controls: []containerControlModel{
					{
						baseControlModel: baseControlModel{
							ID:       types.StringValue("vuln-1"),
							Priority: types.StringValue("HIGH"),
							Disabled: types.BoolValue(false),
						},
					},
				},
			},
		},
	}

	policy, diags := planToAPI(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(policy.FeatureScope) != 1 {
		t.Errorf("expected feature scope, got %+v", policy.FeatureScope)
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
