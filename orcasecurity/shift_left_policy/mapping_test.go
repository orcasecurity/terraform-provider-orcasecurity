package shift_left_policy

import (
	"encoding/json"
	"strings"
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

// Legacy aggregate file_system uses flat policy_data.controls (unlike the scoped
// file_system_* sub-types), and round-trips back into the file_system block.

func TestPlanToAPI_FileSystem_FlatShapeRoundTrip(t *testing.T) {
	model := &shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("file_system"),
		Name:                     types.StringValue("fs policy"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		FileSystem: &controlsBlockModel{
			Controls: []baseControlModel{
				{ID: types.StringValue("fs-1"), Priority: types.StringValue("HIGH"), Disabled: types.BoolValue(false)},
			},
		},
	}

	policy, diags := planToAPI(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	var pd map[string]interface{}
	if err := json.Unmarshal(policy.PolicyData, &pd); err != nil {
		t.Fatalf("policy_data not valid JSON: %v", err)
	}
	if _, ok := pd["controls"].([]interface{}); !ok {
		t.Fatalf("expected flat policy_data.controls, got %v", pd)
	}
	if _, scoped := pd["feature_scope"]; scoped {
		t.Error("legacy file_system must not send feature_scope")
	}

	state := apiToState(&policy, nil)
	if state.FileSystem == nil || len(state.FileSystem.Controls) != 1 {
		t.Fatalf("file_system block did not round-trip, got %+v", state.FileSystem)
	}
}

// Legacy sca round-trips through the licenses block shape.

func TestPlanToAPI_Sca_RoundTrip(t *testing.T) {
	model := &shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("sca"),
		Name:                     types.StringValue("sca policy"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		Sca: &licensesBlockModel{
			Controls: []licenseControlModel{
				{baseControlModel: baseControlModel{ID: types.StringValue("sca-1"), Priority: types.StringValue("HIGH"), Disabled: types.BoolValue(false)}},
			},
		},
	}

	policy, diags := planToAPI(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(policy.Controls) == 0 {
		t.Error("expected controls to be set for sca")
	}
	state := apiToState(&policy, nil)
	if state.Sca == nil || len(state.Sca.Controls) != 1 {
		t.Fatalf("sca block did not round-trip, got %+v", state.Sca)
	}
}

// file_system_* requires scoped policy_data; flat controls rejected (400).

func TestPlanToAPI_FileSystemVulnerabilities_ScopedShape(t *testing.T) {
	model := &shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("file_system_vulnerabilities"),
		Name:                     types.StringValue("fsv policy"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		FileSystemVulnerabilities: &controlsBlockModel{
			Controls: []baseControlModel{
				{
					ID:       types.StringValue("fsv-1"),
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

	var pd map[string]interface{}
	if err := json.Unmarshal(policy.PolicyData, &pd); err != nil {
		t.Fatalf("policy_data not valid JSON: %v", err)
	}
	scopes, ok := pd["feature_scope"].([]interface{})
	if !ok || len(scopes) != 1 || scopes[0] != "vulnerabilities" {
		t.Fatalf("expected feature_scope [vulnerabilities], got %v", pd["feature_scope"])
	}
	scoped, ok := pd["vulnerabilities"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected scoped vulnerabilities section, got %v", pd)
	}
	controls, ok := scoped["controls"].([]interface{})
	if !ok || len(controls) != 1 {
		t.Fatalf("expected one scoped control, got %v", scoped)
	}
	if _, flat := pd["controls"]; flat {
		t.Error("flat policy_data.controls must not be sent (API rejects it)")
	}
}

func TestPlanToAPI_FileSystemSecretDetection_ScopedShape(t *testing.T) {
	model := &shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("file_system_secret_detection"),
		Name:                     types.StringValue("fssd policy"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		FileSystemSecretDetection: &controlsBlockModel{
			Controls: []baseControlModel{
				{ID: types.StringValue("sd-1"), Priority: types.StringValue("LOW"), Disabled: types.BoolValue(true)},
			},
		},
	}

	policy, diags := planToAPI(model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	var pd map[string]interface{}
	if err := json.Unmarshal(policy.PolicyData, &pd); err != nil {
		t.Fatalf("policy_data not valid JSON: %v", err)
	}
	if _, ok := pd["secret_detection"].(map[string]interface{}); !ok {
		t.Fatalf("expected scoped secret_detection section, got %v", pd)
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

func TestAPIToState_FileSystemVulnerabilities_ScopedRead(t *testing.T) {
	apiPolicy := &api_client.ShiftLeftPolicy{
		ID:         "policy-1",
		Type:       "file_system_vulnerabilities",
		PolicyData: []byte(`{"feature_scope":["vulnerabilities"],"vulnerabilities":{"controls":[{"id":"fsv-1","priority":"HIGH","disabled":false}]}}`),
	}

	state := apiToState(apiPolicy, nil)
	if state.FileSystemVulnerabilities == nil || len(state.FileSystemVulnerabilities.Controls) != 1 {
		t.Fatalf("expected one file_system_vulnerabilities control, got %+v", state.FileSystemVulnerabilities)
	}
	if state.FileSystemVulnerabilities.Controls[0].ID.ValueString() != "fsv-1" {
		t.Errorf("expected fsv-1, got %s", state.FileSystemVulnerabilities.Controls[0].ID.ValueString())
	}
}

func TestValidateTypeBlock(t *testing.T) {
	model := &shiftLeftPolicyResourceModel{Sast: &sastBlockModel{}}
	if diags := validateTypeBlock("sast", model); diags.HasError() {
		t.Errorf("expected no error when sast block is present, got %v", diags)
	}

	if diags := validateTypeBlock("sast", &shiftLeftPolicyResourceModel{}); !diags.HasError() {
		t.Error("expected error when sast block is missing")
	}

	if diags := validateTypeBlock("nope", &shiftLeftPolicyResourceModel{}); !diags.HasError() {
		t.Error("expected error for unknown policy type")
	}
}

// A block belonging to another type is never read, so accepting it would silently drop
// whatever the user configured there. Fail the plan instead.

func TestValidateTypeBlock_RejectsForeignBlocks(t *testing.T) {
	model := &shiftLeftPolicyResourceModel{
		Sast: &sastBlockModel{},
		Iac:  &iacBlockModel{},
	}
	diags := validateTypeBlock("sast", model)
	if !diags.HasError() {
		t.Fatal("expected an error when an iac block is set on a sast policy")
	}
	if detail := diags.Errors()[0].Detail(); !strings.Contains(detail, `"iac"`) {
		t.Errorf("error should name the offending block, got: %s", detail)
	}
}

// The message lists every offending block, in a stable order despite map iteration.

func TestValidateTypeBlock_ForeignBlockListIsSorted(t *testing.T) {
	model := &shiftLeftPolicyResourceModel{
		Sast:       &sastBlockModel{},
		Iac:        &iacBlockModel{},
		FileSystem: &controlsBlockModel{},
	}
	for range 5 {
		diags := validateTypeBlock("sast", model)
		if !diags.HasError() {
			t.Fatal("expected an error for the foreign blocks")
		}
		if detail := diags.Errors()[0].Detail(); !strings.Contains(detail, `"file_system", "iac"`) {
			t.Fatalf("expected sorted block names, got: %s", detail)
		}
	}
}

// projects_ids = [] means "detach every project"; refresh must keep the empty set rather
// than collapsing it to null, which would diverge from the configuration forever.
