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

	controlBlocks := []struct {
		name        string
		plan, state interface{}
	}{
		{"iac", plan.Iac, state.Iac},
		{"sast", plan.Sast, state.Sast},
		{"file_system", plan.FileSystem, state.FileSystem},
		{"file_system_vulnerabilities", plan.FileSystemVulnerabilities, state.FileSystemVulnerabilities},
		{"file_system_secret_detection", plan.FileSystemSecretDetection, state.FileSystemSecretDetection},
		{"container_image", plan.ContainerImage, state.ContainerImage},
		{"scm_posture", plan.ScmPosture, state.ScmPosture},
		{"licenses", plan.Licenses, state.Licenses},
		{"sca", plan.Sca, state.Sca},
	}
	for _, cb := range controlBlocks {
		if !reflect.DeepEqual(cb.plan, cb.state) {
			return cb.name, true
		}
	}

	return "", false
}
