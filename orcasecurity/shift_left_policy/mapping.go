package shift_left_policy

import (
	"encoding/json"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringSliceFromTypes(values []types.String) []string {
	result := make([]string, 0, len(values))
	for _, v := range values {
		if !v.IsNull() && !v.IsUnknown() {
			result = append(result, v.ValueString())
		}
	}
	return result
}

// stringSliceFromSet extracts the known string elements from a set attribute,
// returning nil for a null/unknown set.
func stringSliceFromSet(s types.Set) []string {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	elems := s.Elements()
	result := make([]string, 0, len(elems))
	for _, e := range elems {
		if v, ok := e.(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			result = append(result, v.ValueString())
		}
	}
	return result
}

// setFromStringSlice builds a set attribute from ids, returning a null set for
// an empty/nil input so an unattached policy reads as null rather than [].
func setFromStringSlice(values []string) types.Set {
	if len(values) == 0 {
		return types.SetNull(types.StringType)
	}
	elems := make([]attr.Value, len(values))
	for i, v := range values {
		elems[i] = types.StringValue(v)
	}
	return types.SetValueMust(types.StringType, elems)
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func validateTypeBlock(policyType string, model *shiftLeftPolicyResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	hasBlock := func(set bool, name string) {
		if !set {
			diags.AddError("Missing type configuration block", fmt.Sprintf("Policy type %q requires the %q block to be set.", policyType, name))
		}
	}

	switch policyType {
	case "iac":
		hasBlock(model.Iac != nil, "iac")
	case "sast":
		hasBlock(model.Sast != nil, "sast")
	case "file_system":
		hasBlock(model.FileSystem != nil, "file_system")
	case "file_system_vulnerabilities":
		hasBlock(model.FileSystemVulnerabilities != nil, "file_system_vulnerabilities")
	case "file_system_secret_detection":
		hasBlock(model.FileSystemSecretDetection != nil, "file_system_secret_detection")
	case "container_image":
		hasBlock(model.ContainerImage != nil, "container_image")
	case "scm_posture":
		hasBlock(model.ScmPosture != nil, "scm_posture")
	case "licenses":
		hasBlock(model.Licenses != nil, "licenses")
	case "sca":
		hasBlock(model.Sca != nil, "sca")
	case "malicious_packages":
		// No controls, no type block required.
	default:
		diags.AddError("Unsupported policy type", fmt.Sprintf("Unknown policy type %q.", policyType))
	}
	return diags
}

func buildControlsAndData(model *shiftLeftPolicyResourceModel, policy *api_client.ShiftLeftPolicy) ([]map[string]interface{}, map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	policyData := map[string]interface{}{}
	var controls []map[string]interface{}

	switch model.Type.ValueString() {
	case "iac":
		controls = iacControlsToMaps(model.Iac)
		policyData["controls"] = controls
	case "sast":
		controls = sastControlsToMaps(model.Sast)
		policyData["controls"] = controls
	case "file_system":
		controls = controlsBlockToMaps(model.FileSystem)
		policyData["controls"] = controls
	case "file_system_vulnerabilities":
		controls = controlsBlockToMaps(model.FileSystemVulnerabilities)
		policyData["controls"] = controls
	case "file_system_secret_detection":
		controls = controlsBlockToMaps(model.FileSystemSecretDetection)
		policyData["controls"] = controls
	case "container_image":
		controls = buildContainerImageData(model.ContainerImage, policy, policyData)
	case "scm_posture":
		scopeRaw, scmControls, d := buildScmScope(model.ScmPosture)
		diags.Append(d...)
		if diags.HasError() {
			return nil, nil, diags
		}
		policy.Scope = scopeRaw
		controls = scmControls
		policyData["controls"] = controls
	case "licenses":
		controls = licenseControlsToMaps(model.Licenses.Controls)
		policyData["controls"] = controls
	case "sca":
		controls = licenseControlsToMaps(model.Sca.Controls)
		policyData["controls"] = controls
	case "malicious_packages":
		// No controls; policy_data is {}.
	}

	return controls, policyData, diags
}

func encodeJSONField(value interface{}, label string, diags *diag.Diagnostics) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		diags.AddError("Failed to encode "+label, err.Error())
		return nil
	}
	return raw
}

func planToAPI(model *shiftLeftPolicyResourceModel) (api_client.ShiftLeftPolicy, diag.Diagnostics) {
	var diags diag.Diagnostics
	policyType := model.Type.ValueString()
	diags.Append(validateTypeBlock(policyType, model)...)
	if diags.HasError() {
		return api_client.ShiftLeftPolicy{}, diags
	}

	policy := api_client.ShiftLeftPolicy{
		Name:                     model.Name.ValueString(),
		Description:              model.Description.ValueString(),
		Disabled:                 model.Disabled.ValueBool(),
		WarnMode:                 model.WarnMode.ValueBool(),
		PriorityFailureThreshold: model.PriorityFailureThreshold.ValueString(),
		Type:                     policyType,
		ProjectsIds:              stringSliceFromSet(model.ProjectsIds),
	}

	controls, policyData, d := buildControlsAndData(model, &policy)
	diags.Append(d...)
	if diags.HasError() {
		return api_client.ShiftLeftPolicy{}, diags
	}

	if len(controls) > 0 {
		if policy.Controls = encodeJSONField(controls, "controls", &diags); diags.HasError() {
			return api_client.ShiftLeftPolicy{}, diags
		}
	}
	if len(policyData) > 0 {
		if policy.PolicyData = encodeJSONField(policyData, "policy_data", &diags); diags.HasError() {
			return api_client.ShiftLeftPolicy{}, diags
		}
	}

	return policy, diags
}

func boolIsTrue(b types.Bool) bool {
	return !b.IsNull() && !b.IsUnknown() && b.ValueBool()
}

// allControlsScopeKeys returns the scope keys for which the config requested all
// catalog controls. Container_image uses feature scope names; other types use "".
func allControlsScopeKeys(model *shiftLeftPolicyResourceModel) []string {
	if model.Type.ValueString() == "container_image" {
		return containerAllControlsScopes(model.ContainerImage)
	}
	if topLevelAllControlsRequested(model) {
		return []string{""}
	}
	return nil
}

// topLevelAllControlsRequested reports whether a single-block policy type set all_controls.
func topLevelAllControlsRequested(model *shiftLeftPolicyResourceModel) bool {
	switch model.Type.ValueString() {
	case "iac":
		return model.Iac != nil && boolIsTrue(model.Iac.AllControls)
	case "sast":
		return model.Sast != nil && boolIsTrue(model.Sast.AllControls)
	case "file_system":
		return model.FileSystem != nil && boolIsTrue(model.FileSystem.AllControls)
	case "file_system_vulnerabilities":
		return model.FileSystemVulnerabilities != nil && boolIsTrue(model.FileSystemVulnerabilities.AllControls)
	case "file_system_secret_detection":
		return model.FileSystemSecretDetection != nil && boolIsTrue(model.FileSystemSecretDetection.AllControls)
	case "licenses":
		return model.Licenses != nil && boolIsTrue(model.Licenses.AllControls)
	case "sca":
		return model.Sca != nil && boolIsTrue(model.Sca.AllControls)
	}
	return false
}

func containerAllControlsScopes(block *containerImageBlockModel) []string {
	if block == nil {
		return nil
	}
	var keys []string
	scopes := []struct {
		key   string
		block *containerScopeBlockModel
	}{
		{"vulnerabilities", block.Vulnerabilities},
		{"secret_detection", block.SecretDetection},
		{"container_image_best_practices", block.ContainerImageBestPractices},
		{"custom", block.Custom},
	}
	for _, s := range scopes {
		if s.block != nil && boolIsTrue(s.block.AllControls) {
			keys = append(keys, s.key)
		}
	}
	return keys
}

func apiToState(apiPolicy *api_client.ShiftLeftPolicy, existing *shiftLeftPolicyResourceModel) *shiftLeftPolicyResourceModel {
	model := &shiftLeftPolicyResourceModel{
		ID:                       types.StringValue(apiPolicy.ID),
		Type:                     types.StringValue(apiPolicy.Type),
		Name:                     types.StringValue(apiPolicy.Name),
		Description:              types.StringValue(apiPolicy.Description),
		Disabled:                 types.BoolValue(apiPolicy.Disabled),
		WarnMode:                 types.BoolValue(apiPolicy.WarnMode),
		PriorityFailureThreshold: types.StringValue(apiPolicy.PriorityFailureThreshold),
		ProjectsIds:              setFromStringSlice(apiPolicy.ProjectsIds),
		Builtin:                  types.BoolValue(apiPolicy.Builtin),
	}

	policyType := apiPolicy.Type
	if policyType == "" && existing != nil {
		policyType = existing.Type.ValueString()
		model.Type = types.StringValue(policyType)
	}

	policyData := policyDataFromRaw(apiPolicy.PolicyData)
	controls := resolveControls(apiPolicy, policyData)

	applyTypeBlockToState(model, policyType, apiPolicy, policyData, controls)

	if existing != nil {
		mergeStateFromPlan(model, existing)
	}

	return model
}

// stateFromPlanAfterWrite anchors post-create/update state on the applied plan.
// Nested container controls in API responses are not reliably round-tripped immediately after write.
func stateFromPlanAfterWrite(plan *shiftLeftPolicyResourceModel, apiPolicy *api_client.ShiftLeftPolicy) *shiftLeftPolicyResourceModel {
	state := *plan
	state.ID = types.StringValue(apiPolicy.ID)
	state.Builtin = types.BoolValue(apiPolicy.Builtin)
	// projects_ids is Optional+Computed: when the user omitted it the plan value
	// is unknown, so anchor it on the projects the API reports as attached.
	if plan.ProjectsIds.IsUnknown() {
		state.ProjectsIds = setFromStringSlice(apiPolicy.ProjectsIds)
	}
	return &state
}

func stringValueFromMap(m map[string]interface{}, key string) types.String {
	if v, ok := m[key].(string); ok {
		return types.StringValue(v)
	}
	return types.String{}
}

func boolValueFromMap(m map[string]interface{}, key string) types.Bool {
	if v, ok := m[key].(bool); ok {
		return types.BoolValue(v)
	}
	return types.Bool{}
}

func parseImportID(id string) (policyType, policyID string, err error) {
	for i := 0; i < len(id); i++ {
		if id[i] == '/' {
			return id[:i], id[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("import ID must be in the format <type>/<id>, got %q", id)
}
