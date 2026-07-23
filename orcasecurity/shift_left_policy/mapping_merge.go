package shift_left_policy

import "github.com/hashicorp/terraform-plugin-framework/types"

func mergeBaseControlFromPlan(dst *baseControlModel, src baseControlModel) {
	if isStringSet(src.ID) {
		dst.ID = src.ID
	} else {
		// Config referenced the control by title (or it is custom): keep id null
		// in state so an API-resolved id does not show as drift.
		dst.ID = types.StringNull()
	}
	if isStringSet(src.Priority) {
		dst.Priority = src.Priority
	}
	if !src.Disabled.IsNull() && !src.Disabled.IsUnknown() {
		dst.Disabled = src.Disabled
	}
	if !isStringSet(src.Title) {
		dst.Title = types.StringNull()
	}
	if src.Conditions == nil {
		dst.Conditions = nil
	}
}

func mergeControlsBlockFromPlan(dst, src *controlsBlockModel) {
	if dst == nil || src == nil {
		return
	}
	dst.AllControls = src.AllControls
	if boolIsTrue(src.AllControls) {
		dst.Controls = nil
		return
	}
	for i := range dst.Controls {
		if i < len(src.Controls) {
			mergeBaseControlFromPlan(&dst.Controls[i], src.Controls[i])
		}
	}
}

func mergeIacBlockFromPlan(dst, src *iacBlockModel) {
	if dst == nil || src == nil {
		return
	}
	dst.AllControls = src.AllControls
	if boolIsTrue(src.AllControls) {
		dst.Controls = nil
		return
	}
	for i := range dst.Controls {
		if i >= len(src.Controls) {
			continue
		}
		mergeBaseControlFromPlan(&dst.Controls[i].baseControlModel, src.Controls[i].baseControlModel)
		dst.Controls[i].Frameworks = src.Controls[i].Frameworks
		dst.Controls[i].OrcaAlertRuleType = src.Controls[i].OrcaAlertRuleType
	}
}

func mergeContainerScopeFromPlan(dst, src *containerScopeBlockModel) {
	if dst == nil || src == nil {
		return
	}
	dst.AllControls = src.AllControls
	if boolIsTrue(src.AllControls) {
		dst.Controls = nil
		return
	}
	for i := range src.Controls {
		if i >= len(dst.Controls) {
			break
		}
		mergeBaseControlFromPlan(&dst.Controls[i].baseControlModel, src.Controls[i].baseControlModel)
		dst.Controls[i].Origin = src.Controls[i].Origin
	}
}

func mergeSastExtrasFromPlan(dst *sastControlModel, src sastControlModel) {
	dst.Languages = src.Languages
	dst.Owasp = src.Owasp
	dst.Cwe = src.Cwe
	dst.Section = src.Section
	dst.Confidence = src.Confidence
	dst.Impact = src.Impact
	dst.Likelihood = src.Likelihood
}

func mergeLicenseExtrasFromPlan(dst *licenseControlModel, src licenseControlModel) {
	dst.LicenseID = src.LicenseID
	dst.LicenseCategory = src.LicenseCategory
	dst.IsOsiApproved = src.IsOsiApproved
	dst.IsDeprecated = src.IsDeprecated
	dst.IsFsfLibre = src.IsFsfLibre
	dst.Url = src.Url
	dst.AdditionalInfo = src.AdditionalInfo
}

func mergeContainerImageFromPlan(dst, src *containerImageBlockModel) {
	if dst == nil || src == nil {
		return
	}
	mergeContainerScopeFromPlan(dst.Vulnerabilities, src.Vulnerabilities)
	mergeContainerScopeFromPlan(dst.SecretDetection, src.SecretDetection)
	mergeContainerScopeFromPlan(dst.ContainerImageBestPractices, src.ContainerImageBestPractices)
	mergeContainerScopeFromPlan(dst.Custom, src.Custom)
}

func mergeSastBlockFromPlan(dst, src *sastBlockModel) {
	if dst == nil || src == nil {
		return
	}
	dst.AllControls = src.AllControls
	if boolIsTrue(src.AllControls) {
		dst.Controls = nil
		return
	}
	for i := range dst.Controls {
		if i >= len(src.Controls) {
			continue
		}
		mergeBaseControlFromPlan(&dst.Controls[i].baseControlModel, src.Controls[i].baseControlModel)
		mergeSastExtrasFromPlan(&dst.Controls[i], src.Controls[i])
	}
}

func mergeLicensesBlockFromPlan(dst, src *licensesBlockModel) {
	if dst == nil || src == nil {
		return
	}
	dst.AllControls = src.AllControls
	if boolIsTrue(src.AllControls) {
		dst.Controls = nil
		return
	}
	for i := range dst.Controls {
		if i >= len(src.Controls) {
			continue
		}
		mergeBaseControlFromPlan(&dst.Controls[i].baseControlModel, src.Controls[i].baseControlModel)
		mergeLicenseExtrasFromPlan(&dst.Controls[i], src.Controls[i])
	}
}

func mergeStateFromPlan(state, plan *shiftLeftPolicyResourceModel) {
	switch plan.Type.ValueString() {
	case "iac":
		mergeIacBlockFromPlan(state.Iac, plan.Iac)
	case "sast":
		mergeSastBlockFromPlan(state.Sast, plan.Sast)
	case "file_system":
		mergeControlsBlockFromPlan(state.FileSystem, plan.FileSystem)
	case "file_system_vulnerabilities":
		mergeControlsBlockFromPlan(state.FileSystemVulnerabilities, plan.FileSystemVulnerabilities)
	case "file_system_secret_detection":
		mergeControlsBlockFromPlan(state.FileSystemSecretDetection, plan.FileSystemSecretDetection)
	case "container_image":
		mergeContainerImageFromPlan(state.ContainerImage, plan.ContainerImage)
	case "licenses":
		mergeLicensesBlockFromPlan(state.Licenses, plan.Licenses)
	case "sca":
		mergeLicensesBlockFromPlan(state.Sca, plan.Sca)
	}
}
