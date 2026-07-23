package shift_left_policy

import "reflect"

// builtinNonProjectFieldChanged reports the first managed field other than
// projects_ids that differs between plan and state. The UI allows built-in
// policies to change only their attached projects via the same PUT; every
// other field is read-only for built-ins, including every control block
// (iac/sast/file_system/.../sca) — changing a control inside any of those
// blocks must be rejected just like changing name/description/etc.
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

	// Compare every type block via the shared handler table (policyTypes keeps
	// the iteration order deterministic so the reported field name is stable).
	for _, name := range policyTypes {
		h := policyTypeHandlers[name]
		if h.block == nil {
			continue
		}
		if !reflect.DeepEqual(h.block(plan), h.block(state)) {
			return name, true
		}
	}

	return "", false
}
