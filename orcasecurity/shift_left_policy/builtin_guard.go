package shift_left_policy

import "reflect"

// Built-in locked fields: name; container_image.feature_scope. The built-in
// scm_posture policy never reaches this resource (it is owned exclusively by
// shift_left_scm_posture_default_policy, and importing it here is rejected).
func builtinLockedFieldChanged(plan, state *shiftLeftPolicyResourceModel) (string, bool) {
	if !plan.Name.Equal(state.Name) {
		return "name", true
	}
	if plan.Type.ValueString() == "container_image" &&
		plan.ContainerImage != nil && state.ContainerImage != nil &&
		!reflect.DeepEqual(plan.ContainerImage.FeatureScope, state.ContainerImage.FeatureScope) {
		return "container_image.feature_scope", true
	}
	return "", false
}
