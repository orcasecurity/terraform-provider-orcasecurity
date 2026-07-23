package shift_left_policy

import "reflect"

// builtinLockedFieldChanged reports the first field that the API forbids
// changing on a built-in policy, mirroring the server contract
// (typed_policy_view / scm_posture_policy_service in shiftleft-core):
//
//   - every built-in: `name` is immutable (and deletion is blocked, enforced
//     separately in Delete);
//   - container_image built-ins: `feature_scope` is additionally immutable;
//   - scm_posture built-ins: `description` and `scope` are additionally
//     immutable (they are org-global policies).
//
// Everything else — description (non-scm_posture), disabled, warn_mode,
// priority_failure_threshold, control overrides, and projects_ids — is
// updatable on built-ins, matching what the API accepts and the UI exposes.
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
