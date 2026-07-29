package shift_left_policy

import "github.com/hashicorp/terraform-plugin-framework/types"

func isStringSet(value types.String) bool {
	return !value.IsNull() && !value.IsUnknown() && value.ValueString() != ""
}

func mergeBaseControlFromPlan(dst *baseControlModel, src baseControlModel) {
	if isStringSet(src.ID) {
		dst.ID = src.ID
	} else {
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

// mergeControlBlock restores the all_controls flag and index-wise merges the
// config-owned control fields from prior state. dst is fresh from the API, so
// controls it added/removed out of band still surface. Shared by every flat block.
func mergeControlBlock[C any](dstAll *types.Bool, srcAll types.Bool, dstControls *[]C, srcControls []C, mergeControl func(dst *C, src C)) {
	*dstAll = srcAll
	if boolIsTrue(srcAll) {
		*dstControls = nil
		return
	}
	d := *dstControls
	for i := range d {
		if i < len(srcControls) {
			mergeControl(&d[i], srcControls[i])
		}
	}
}

func mergeIacControlFromPlan(dst *iacControlModel, src iacControlModel) {
	mergeBaseControlFromPlan(&dst.baseControlModel, src.baseControlModel)
	dst.Frameworks = src.Frameworks
	dst.OrcaAlertRuleType = src.OrcaAlertRuleType
}

func mergeContainerControlFromPlan(dst *containerControlModel, src containerControlModel) {
	mergeBaseControlFromPlan(&dst.baseControlModel, src.baseControlModel)
	dst.Origin = src.Origin
}

func mergeSastControlFromPlan(dst *sastControlModel, src sastControlModel) {
	mergeBaseControlFromPlan(&dst.baseControlModel, src.baseControlModel)
	dst.Languages = src.Languages
	dst.Owasp = src.Owasp
	dst.Cwe = src.Cwe
	dst.Section = src.Section
	dst.Confidence = src.Confidence
	dst.Impact = src.Impact
	dst.Likelihood = src.Likelihood
}

func mergeLicenseControlFromPlan(dst *licenseControlModel, src licenseControlModel) {
	mergeBaseControlFromPlan(&dst.baseControlModel, src.baseControlModel)
	dst.LicenseID = src.LicenseID
	dst.LicenseCategory = src.LicenseCategory
	dst.IsOsiApproved = src.IsOsiApproved
	dst.IsDeprecated = src.IsDeprecated
	dst.IsFsfLibre = src.IsFsfLibre
	dst.Url = src.Url
	dst.AdditionalInfo = src.AdditionalInfo
}

func mergeControlsBlockFromPlan(dst, src *controlsBlockModel) {
	if dst == nil || src == nil {
		return
	}
	mergeControlBlock(&dst.AllControls, src.AllControls, &dst.Controls, src.Controls, mergeBaseControlFromPlan)
}

func mergeIacBlockFromPlan(dst, src *iacBlockModel) {
	if dst == nil || src == nil {
		return
	}
	mergeControlBlock(&dst.AllControls, src.AllControls, &dst.Controls, src.Controls, mergeIacControlFromPlan)
}

func mergeContainerScopeFromPlan(dst, src *containerScopeBlockModel) {
	if dst == nil || src == nil {
		return
	}
	mergeControlBlock(&dst.AllControls, src.AllControls, &dst.Controls, src.Controls, mergeContainerControlFromPlan)
}

func mergeSastBlockFromPlan(dst, src *sastBlockModel) {
	if dst == nil || src == nil {
		return
	}
	mergeControlBlock(&dst.AllControls, src.AllControls, &dst.Controls, src.Controls, mergeSastControlFromPlan)
}

func mergeLicensesBlockFromPlan(dst, src *licensesBlockModel) {
	if dst == nil || src == nil {
		return
	}
	mergeControlBlock(&dst.AllControls, src.AllControls, &dst.Controls, src.Controls, mergeLicenseControlFromPlan)
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

// All per-control fields are config-owned: the API enriches scm/entity/threat
// from the control catalog, so keeping the API values would drift against a
// config that omits them. Take them from state; the index-wise loop still
// surfaces controls the API added or removed.
func mergeScmControlFromPlan(dst *scmControlModel, src scmControlModel) {
	dst.ID = src.ID
	dst.Priority = src.Priority
	dst.Disabled = src.Disabled
	dst.Scm = src.Scm
	dst.Entity = src.Entity
	dst.Threat = src.Threat
}

func mergeScmPostureBlockFromPlan(dst, src *scmPostureBlockModel) {
	if dst == nil || src == nil {
		return
	}
	for i := range dst.Controls {
		if i < len(src.Controls) {
			mergeScmControlFromPlan(&dst.Controls[i], src.Controls[i])
		}
	}
	// Scope is an unordered map on the wire; keep config order from state to avoid a diff.
	if len(src.Scope) > 0 {
		dst.Scope = src.Scope
	}
}
