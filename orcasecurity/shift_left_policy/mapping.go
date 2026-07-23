package shift_left_policy

import (
	"encoding/json"
	"fmt"
	"strings"

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
	if apiPolicy.PriorityFailureThreshold == "" && existing != nil &&
		!existing.PriorityFailureThreshold.IsNull() && !existing.PriorityFailureThreshold.IsUnknown() {
		model.PriorityFailureThreshold = existing.PriorityFailureThreshold
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
	policyType, policyID, ok := strings.Cut(id, "/")
	if !ok || policyType == "" || policyID == "" {
		return "", "", fmt.Errorf("import ID must be in the format <type>/<id>, got %q", id)
	}
	return policyType, policyID, nil
}
