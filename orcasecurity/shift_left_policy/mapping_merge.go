package shift_left_policy

import (
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func mergeBaseControlFromPlan(dst *baseControlModel, src baseControlModel) {
	if tfconv.StringIsSet(src.ID) {
		dst.ID = src.ID
	} else {
		dst.ID = types.StringNull()
	}
	if tfconv.StringIsSet(src.Priority) {
		dst.Priority = src.Priority
	}
	if tfconv.Known(src.Disabled) {
		dst.Disabled = src.Disabled
	}
	if !tfconv.StringIsSet(src.Title) {
		dst.Title = types.StringNull()
	}
	if src.Conditions == nil {
		dst.Conditions = nil
	}
}

func (c *baseControlModel) controlID() string { return c.ID.ValueString() }

func (c *scmControlModel) controlID() string { return c.ID.ValueString() }

// mergeControlBlock overlays the prior-state controls onto the ones the API returned, so controls
// the API added or removed still surface as drift.
func mergeControlBlock[C any](dstAll *types.Bool, srcAll types.Bool, dstControls *[]C, srcControls []C,
	controlID func(*C) string, mergeControl func(dst *C, src C)) {
	*dstAll = srcAll
	if tfconv.BoolIsTrue(srcAll) {
		*dstControls = nil
		return
	}
	apiControls := *dstControls
	for i, prior := range pairControls(apiControls, srcControls, controlID) {
		if prior != nil {
			mergeControl(&apiControls[i], *prior)
		}
	}
}

// pairControls lines each API control up with the prior-state control it belongs to. The control id
// is the only stable identity: the API is free to reorder or drop controls, and pairing by position
// would then copy one control's priority/disabled override onto an unrelated control. An id is
// optional in configuration, so a list that does not carry ids throughout still falls back to
// position — which is no worse than the pairing such a list had before.
func pairControls[C any](apiControls, prior []C, controlID func(*C) string) []*C {
	paired := make([]*C, len(apiControls))
	if byID, ok := indexControlsByID(apiControls, prior, controlID); ok {
		for i := range apiControls {
			paired[i] = byID[controlID(&apiControls[i])]
		}
		return paired
	}
	for i := range apiControls {
		if i < len(prior) {
			paired[i] = &prior[i]
		}
	}
	return paired
}

// indexControlsByID keys the prior controls by id, reporting false when either side fails to
// identify every control by a unique id and pairing therefore cannot rely on ids.
func indexControlsByID[C any](apiControls, prior []C, controlID func(*C) string) (map[string]*C, bool) {
	byID := make(map[string]*C, len(prior))
	for i := range prior {
		id := controlID(&prior[i])
		if _, taken := byID[id]; id == "" || taken {
			return nil, false
		}
		byID[id] = &prior[i]
	}
	for i := range apiControls {
		if controlID(&apiControls[i]) == "" {
			return nil, false
		}
	}
	return byID, true
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
	mergeControlBlock(&dst.AllControls, src.AllControls, &dst.Controls, src.Controls,
		(*baseControlModel).controlID, mergeBaseControlFromPlan)
}

func mergeIacBlockFromPlan(dst, src *iacBlockModel) {
	if dst == nil || src == nil {
		return
	}
	mergeControlBlock(&dst.AllControls, src.AllControls, &dst.Controls, src.Controls,
		(*iacControlModel).controlID, mergeIacControlFromPlan)
}

func mergeContainerScopeFromPlan(dst, src *containerScopeBlockModel) {
	if dst == nil || src == nil {
		return
	}
	mergeControlBlock(&dst.AllControls, src.AllControls, &dst.Controls, src.Controls,
		(*containerControlModel).controlID, mergeContainerControlFromPlan)
}

func mergeSastBlockFromPlan(dst, src *sastBlockModel) {
	if dst == nil || src == nil {
		return
	}
	mergeControlBlock(&dst.AllControls, src.AllControls, &dst.Controls, src.Controls,
		(*sastControlModel).controlID, mergeSastControlFromPlan)
}

func mergeLicensesBlockFromPlan(dst, src *licensesBlockModel) {
	if dst == nil || src == nil {
		return
	}
	mergeControlBlock(&dst.AllControls, src.AllControls, &dst.Controls, src.Controls,
		(*licenseControlModel).controlID, mergeLicenseControlFromPlan)
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

// Scm/Entity/Threat are catalog-computed: always from state. Priority/Disabled: from state when set, else keep API (surfaces OOB drift).
func mergeScmControlFromPlan(dst *scmControlModel, src scmControlModel) {
	if tfconv.StringIsSet(src.ID) {
		dst.ID = src.ID
	} else {
		dst.ID = types.StringNull()
	}
	if tfconv.StringIsSet(src.Priority) {
		dst.Priority = src.Priority
	}
	if tfconv.Known(src.Disabled) {
		dst.Disabled = src.Disabled
	}
	dst.Scm = src.Scm
	dst.Entity = src.Entity
	dst.Threat = src.Threat
}

func mergeScmPostureBlockFromPlan(dst, src *scmPostureBlockModel) {
	if dst == nil || src == nil {
		return
	}
	for i, prior := range pairControls(dst.Controls, src.Controls, (*scmControlModel).controlID) {
		if prior != nil {
			mergeScmControlFromPlan(&dst.Controls[i], *prior)
		}
	}
	// Scope stays from API so OOB scope changes surface as drift.
}
