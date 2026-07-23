package shift_left_policy

import "reflect"

// Built-in locked fields: name; container_image.feature_scope; scm_posture description/scope.
func builtinLockedFieldChanged(plan, state *shiftLeftPolicyResourceModel) (string, bool) {
	if !plan.Name.Equal(state.Name) {
		return "name", true
	}
	switch plan.Type.ValueString() {
	case "container_image":
		if plan.ContainerImage != nil && state.ContainerImage != nil &&
			!reflect.DeepEqual(plan.ContainerImage.FeatureScope, state.ContainerImage.FeatureScope) {
			return "container_image.feature_scope", true
		}
	case "scm_posture":
		if !plan.Description.Equal(state.Description) {
			return "description", true
		}
		planScope := scmScopeOf(plan)
		stateScope := scmScopeOf(state)
		if !reflect.DeepEqual(planScope, stateScope) {
			return "scm_posture.scope", true
		}
	}
	return "", false
}

func scmScopeOf(m *shiftLeftPolicyResourceModel) []scmScopeEntryModel {
	if m.ScmPosture == nil {
		return nil
	}
	return m.ScmPosture.Scope
}
