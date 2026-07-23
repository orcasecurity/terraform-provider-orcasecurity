package shift_left_policy

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
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
		ID:       "policy-1",
		Type:     "container_image",
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
		ID:       "policy-1",
		Type:     "iac",
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

// TestAPIToState_ProjectsIdsPopulatedFromInstance is the import-path case
// (Step 5 of task-1b): once GetShiftLeftPolicy populates
// api_client.ShiftLeftPolicy.ProjectsIds from the `projects` array (see
// shift_left_policy.go's populateProjectsIds), apiToState must carry those
// ids into the model. ImportState calls apiToState(instance, nil), so no
// prior state exists to merge against.
func TestAPIToState_ProjectsIdsPopulatedFromInstance(t *testing.T) {
	apiPolicy := &api_client.ShiftLeftPolicy{
		ID:          "policy-1",
		Type:        "licenses",
		Builtin:     true,
		ProjectsIds: []string{"a", "b"},
		PolicyData:  []byte(`{"controls":[]}`),
	}

	state := apiToState(apiPolicy, nil)
	if len(state.ProjectsIds) != 2 {
		t.Fatalf("expected 2 projects_ids, got %#v", state.ProjectsIds)
	}
	if state.ProjectsIds[0].ValueString() != "a" || state.ProjectsIds[1].ValueString() != "b" {
		t.Errorf("expected [a b], got %#v", state.ProjectsIds)
	}
}

// TestAPIToState_ProjectsIdsAuthoritativeOnRead is the WASP-1483 task-1d fix:
// on Read/refresh, projects_ids must reflect the API even when prior state
// had none -- otherwise out-of-band attach/detach never surfaces as drift.
// This replaces the non-asserting TestAPIToState_ProjectsIdsRefreshEntanglement
// pseudo-test left by task-1b.
func TestAPIToState_ProjectsIdsAuthoritativeOnRead(t *testing.T) {
	// prior state had NO projects; API now reports two attached (out-of-band attach)
	existing := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("licenses"), ProjectsIds: nil,
	}
	api := &api_client.ShiftLeftPolicy{
		ID: "p1", Type: "licenses", ProjectsIds: []string{"proj-a", "proj-b"},
	}
	state := apiToState(api, existing)
	got := stringSliceFromTypes(state.ProjectsIds)
	if len(got) != 2 {
		t.Fatalf("expected refresh to reflect API projects [proj-a proj-b], got %v", got)
	}
}

// TestAPIToState_ProjectsIdsEmptyStaysNull ensures an unattached policy does
// not churn null<->[] across reads.
func TestAPIToState_ProjectsIdsEmptyStaysNull(t *testing.T) {
	// unattached policy: API returns none -> null (no null-vs-[] churn)
	existing := &shiftLeftPolicyResourceModel{Type: types.StringValue("licenses")}
	api := &api_client.ShiftLeftPolicy{ID: "p1", Type: "licenses"}
	state := apiToState(api, existing)
	if len(state.ProjectsIds) != 0 {
		t.Fatalf("expected nil/empty, got %v", state.ProjectsIds)
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

func TestPlanToAPI_Sast(t *testing.T) {
	model := &shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("sast"),
		Name:                     types.StringValue("sast policy"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		Sast: &sastBlockModel{
			Controls: []sastControlModel{
				{
					baseControlModel: baseControlModel{
						ID:       types.StringValue("sast-1"),
						Priority: types.StringValue("HIGH"),
						Disabled: types.BoolValue(false),
					},
					Languages: []types.String{types.StringValue("python")},
					Owasp:     []types.String{types.StringValue("A01")},
				},
			},
		},
	}

	policy, diags := planToAPI(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(policy.Controls) == 0 || len(policy.PolicyData) == 0 {
		t.Error("expected controls and policy_data to be set for sast")
	}
}

func TestPlanToAPI_Licenses(t *testing.T) {
	model := &shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("licenses"),
		Name:                     types.StringValue("license policy"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		Licenses: &licensesBlockModel{
			Controls: []licenseControlModel{
				{
					baseControlModel: baseControlModel{
						ID:       types.StringValue("lic-1"),
						Priority: types.StringValue("HIGH"),
						Disabled: types.BoolValue(true),
					},
					LicenseCategory: types.StringValue("copyleft"),
				},
			},
		},
	}

	policy, diags := planToAPI(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(policy.Controls) == 0 {
		t.Error("expected controls to be set for licenses")
	}
}

func TestPlanToAPI_FileSystem(t *testing.T) {
	model := &shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("file_system"),
		Name:                     types.StringValue("fs policy"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		FileSystem: &controlsBlockModel{
			Controls: []baseControlModel{
				{
					ID:       types.StringValue("fs-1"),
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
	if len(policy.Controls) == 0 {
		t.Error("expected controls to be set for file_system")
	}
}

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

func TestPlanToAPI_UnsupportedType(t *testing.T) {
	model := &shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("bogus"),
		Name:                     types.StringValue("x"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
	}
	_, diags := planToAPI(model)
	if !diags.HasError() {
		t.Fatal("expected error for unsupported policy type")
	}
}

func TestAPIToState_Sast(t *testing.T) {
	apiPolicy := &api_client.ShiftLeftPolicy{
		ID:         "policy-1",
		Type:       "sast",
		Controls:   []byte(`[{"id":"sast-1","priority":"HIGH","disabled":true,"languages":["python"],"section":"injection"}]`),
		PolicyData: []byte(`{"controls":[{"id":"sast-1","priority":"HIGH","disabled":true,"languages":["python"],"section":"injection"}]}`),
	}

	state := apiToState(apiPolicy, nil)
	if state.Sast == nil || len(state.Sast.Controls) != 1 {
		t.Fatalf("expected one sast control, got %+v", state.Sast)
	}
	ctrl := state.Sast.Controls[0]
	if ctrl.ID.ValueString() != "sast-1" {
		t.Errorf("expected sast-1, got %s", ctrl.ID.ValueString())
	}
	if ctrl.Section.ValueString() != "injection" {
		t.Errorf("expected section injection, got %s", ctrl.Section.ValueString())
	}
	if len(ctrl.Languages) != 1 || ctrl.Languages[0].ValueString() != "python" {
		t.Errorf("expected languages [python], got %+v", ctrl.Languages)
	}
}

func TestAPIToState_Licenses(t *testing.T) {
	apiPolicy := &api_client.ShiftLeftPolicy{
		ID:       "policy-1",
		Type:     "licenses",
		Controls: []byte(`[{"id":"lic-1","priority":"HIGH","disabled":true,"license_category":"copyleft","is_osi_approved":true}]`),
	}

	state := apiToState(apiPolicy, nil)
	if state.Licenses == nil || len(state.Licenses.Controls) != 1 {
		t.Fatalf("expected one license control, got %+v", state.Licenses)
	}
	ctrl := state.Licenses.Controls[0]
	if ctrl.LicenseCategory.ValueString() != "copyleft" {
		t.Errorf("expected copyleft, got %s", ctrl.LicenseCategory.ValueString())
	}
	if !ctrl.IsOsiApproved.ValueBool() {
		t.Error("expected is_osi_approved true")
	}
}

func TestAPIToState_FileSystem(t *testing.T) {
	apiPolicy := &api_client.ShiftLeftPolicy{
		ID:       "policy-1",
		Type:     "file_system",
		Controls: []byte(`[{"id":"fs-1","priority":"HIGH","disabled":false}]`),
	}

	state := apiToState(apiPolicy, nil)
	if state.FileSystem == nil || len(state.FileSystem.Controls) != 1 {
		t.Fatalf("expected one file_system control, got %+v", state.FileSystem)
	}
	if state.FileSystem.Controls[0].ID.ValueString() != "fs-1" {
		t.Errorf("expected fs-1, got %s", state.FileSystem.Controls[0].ID.ValueString())
	}
}

func TestValidateTypeBlock(t *testing.T) {
	// Matching block present: no error.
	model := &shiftLeftPolicyResourceModel{Sast: &sastBlockModel{}}
	if diags := validateTypeBlock("sast", model); diags.HasError() {
		t.Errorf("expected no error when sast block is present, got %v", diags)
	}

	// Missing block: error.
	if diags := validateTypeBlock("sast", &shiftLeftPolicyResourceModel{}); !diags.HasError() {
		t.Error("expected error when sast block is missing")
	}

	// Unknown type: error.
	if diags := validateTypeBlock("nope", &shiftLeftPolicyResourceModel{}); !diags.HasError() {
		t.Error("expected error for unknown policy type")
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

func TestMergeStateFromPlan_AllControlsClearsControls(t *testing.T) {
	state := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("iac"),
		Iac: &iacBlockModel{
			Controls: []iacControlModel{
				{baseControlModel: baseControlModel{ID: types.StringValue("from-api")}},
			},
		},
	}
	plan := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("iac"),
		Iac:  &iacBlockModel{AllControls: types.BoolValue(true)},
	}

	mergeStateFromPlan(state, plan)
	if !state.Iac.AllControls.ValueBool() {
		t.Error("expected all_controls to be carried from plan")
	}
	if len(state.Iac.Controls) != 0 {
		t.Errorf("expected controls cleared when all_controls is set, got %+v", state.Iac.Controls)
	}
}

func TestMergeStateFromPlan_TitleReferenceKeepsIDNull(t *testing.T) {
	state := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("iac"),
		Iac: &iacBlockModel{
			Controls: []iacControlModel{
				{baseControlModel: baseControlModel{
					ID:       types.StringValue("api-resolved-id"),
					Priority: types.StringValue("HIGH"),
					Disabled: types.BoolValue(true),
				}},
			},
		},
	}
	plan := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("iac"),
		Iac: &iacBlockModel{
			Controls: []iacControlModel{
				{baseControlModel: baseControlModel{
					Title:    types.StringValue("Some control title"),
					Priority: types.StringValue("HIGH"),
					Disabled: types.BoolValue(true),
				}},
			},
		},
	}

	mergeStateFromPlan(state, plan)
	if !state.Iac.Controls[0].ID.IsNull() {
		t.Errorf("expected id to stay null when plan referenced control by title, got %s", state.Iac.Controls[0].ID.ValueString())
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
