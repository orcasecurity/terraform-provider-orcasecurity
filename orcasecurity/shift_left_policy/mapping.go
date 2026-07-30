package shift_left_policy

import (
	"encoding/json"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringSliceFromTypes(values []types.String) []string {
	result := make([]string, 0, len(values))
	for _, v := range values {
		if tfconv.Known(v) {
			result = append(result, v.ValueString())
		}
	}
	return result
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
		ProjectsIds:              tfconv.SetToStringSlice(model.ProjectsIds),
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

func apiToState(apiPolicy *api_client.ShiftLeftPolicy, existing *shiftLeftPolicyResourceModel) *shiftLeftPolicyResourceModel {
	model := &shiftLeftPolicyResourceModel{
		ID:                       types.StringValue(apiPolicy.ID),
		Type:                     types.StringValue(apiPolicy.Type),
		Name:                     types.StringValue(apiPolicy.Name),
		Description:              tfconv.StringOrNull(apiPolicy.Description),
		Disabled:                 types.BoolValue(apiPolicy.Disabled),
		WarnMode:                 types.BoolValue(apiPolicy.WarnMode),
		PriorityFailureThreshold: types.StringValue(apiPolicy.PriorityFailureThreshold),
		ProjectsIds:              tfconv.StringSliceToSet(apiPolicy.ProjectsIds),
		Builtin:                  types.BoolValue(apiPolicy.Builtin),
	}
	// StringSliceToSet maps no projects to null. Detaching every project is configured as
	// projects_ids = [], so collapsing to null there would leave the config permanently
	// diverged from state; keep the empty set the caller asked for.
	if len(apiPolicy.ProjectsIds) == 0 && existing != nil && tfconv.Known(existing.ProjectsIds) {
		model.ProjectsIds = types.SetValueMust(types.StringType, nil)
	}
	if apiPolicy.PriorityFailureThreshold == "" && existing != nil &&
		tfconv.Known(existing.PriorityFailureThreshold) {
		model.PriorityFailureThreshold = existing.PriorityFailureThreshold
	}
	// description is Optional-only: a configuration that omits it holds null, while the API reports
	// no description as an empty string, so mapping "" to null keeps the two comparable. A caller who
	// explicitly configured "" gets that empty string back rather than a permanent diff to null.
	if apiPolicy.Description == "" && existing != nil && tfconv.Known(existing.Description) {
		model.Description = existing.Description
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
		state.ProjectsIds = tfconv.StringSliceToSet(apiPolicy.ProjectsIds)
	}
	return &state
}
