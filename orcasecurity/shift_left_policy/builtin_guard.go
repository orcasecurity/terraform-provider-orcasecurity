package shift_left_policy

// builtinNonProjectFieldChanged reports the first managed field other than
// projects_ids that differs between plan and state. The UI allows built-in
// policies to change only their attached projects via the same PUT; every
// other field is read-only for built-ins.
func builtinNonProjectFieldChanged(plan, state *shiftLeftPolicyResourceModel) (string, bool) {
	switch {
	case !plan.Name.Equal(state.Name):
		return "name", true
	case !plan.Description.Equal(state.Description):
		return "description", true
	case !plan.Disabled.Equal(state.Disabled):
		return "disabled", true
	case !plan.WarnMode.Equal(state.WarnMode):
		return "warn_mode", true
	case !plan.PriorityFailureThreshold.Equal(state.PriorityFailureThreshold):
		return "priority_failure_threshold", true
	}
	return "", false
}
