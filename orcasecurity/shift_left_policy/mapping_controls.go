package shift_left_policy

import (
	"encoding/json"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func conditionsToMap(c *conditionsModel) map[string]interface{} {
	if c == nil {
		return nil
	}
	result := map[string]interface{}{}
	if !c.FixAvailable.IsNull() {
		result["fix_available"] = c.FixAvailable.ValueBool()
	}
	if !c.FromBaseImage.IsNull() {
		result["from_base_image"] = c.FromBaseImage.ValueBool()
	}
	if !c.DaysFromDiscovery.IsNull() {
		result["days_from_discovery"] = c.DaysFromDiscovery.ValueInt64()
	}
	if !c.DaysFromFix.IsNull() {
		result["days_from_fix"] = c.DaysFromFix.ValueInt64()
	}
	if !c.HasExploit.IsNull() {
		result["has_exploit"] = c.HasExploit.ValueBool()
	}
	if !c.SeveritiesOperator.IsNull() || len(c.SeveritiesValues) > 0 {
		severities := map[string]interface{}{}
		if !c.SeveritiesOperator.IsNull() {
			severities["operator"] = c.SeveritiesOperator.ValueString()
		}
		if len(c.SeveritiesValues) > 0 {
			severities["values"] = stringSliceFromTypes(c.SeveritiesValues)
		}
		if len(severities) > 0 {
			result["severities"] = severities
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func baseControlToMap(c baseControlModel) map[string]interface{} {
	m := map[string]interface{}{
		"priority": c.Priority.ValueString(),
		"disabled": c.Disabled.ValueBool(),
	}
	if id := c.ID.ValueString(); id != "" {
		m["id"] = id
	}
	if !c.Title.IsNull() && c.Title.ValueString() != "" {
		m["title"] = c.Title.ValueString()
	}
	if cond := conditionsToMap(c.Conditions); cond != nil {
		m["conditions"] = cond
	}
	return m
}

func iacControlToMap(c iacControlModel) map[string]interface{} {
	m := baseControlToMap(c.baseControlModel)
	if len(c.Frameworks) > 0 {
		m["frameworks"] = stringSliceFromTypes(c.Frameworks)
	}
	if !c.OrcaAlertRuleType.IsNull() && c.OrcaAlertRuleType.ValueString() != "" {
		m["orca_alert_rule_type"] = c.OrcaAlertRuleType.ValueString()
	}
	return m
}

func sastControlToMap(c sastControlModel) map[string]interface{} {
	m := baseControlToMap(c.baseControlModel)
	if len(c.Languages) > 0 {
		m["languages"] = stringSliceFromTypes(c.Languages)
	}
	if len(c.Owasp) > 0 {
		m["owasp"] = stringSliceFromTypes(c.Owasp)
	}
	if len(c.Cwe) > 0 {
		m["cwe"] = stringSliceFromTypes(c.Cwe)
	}
	if !c.Section.IsNull() && c.Section.ValueString() != "" {
		m["section"] = c.Section.ValueString()
	}
	if !c.Confidence.IsNull() && c.Confidence.ValueString() != "" {
		m["confidence"] = c.Confidence.ValueString()
	}
	if !c.Impact.IsNull() && c.Impact.ValueString() != "" {
		m["impact"] = c.Impact.ValueString()
	}
	if !c.Likelihood.IsNull() && c.Likelihood.ValueString() != "" {
		m["likelihood"] = c.Likelihood.ValueString()
	}
	return m
}

func containerControlToMap(c containerControlModel) map[string]interface{} {
	m := baseControlToMap(c.baseControlModel)
	if !c.Origin.IsNull() && c.Origin.ValueString() != "" {
		m["origin"] = c.Origin.ValueString()
	}
	return m
}

func licenseControlToMap(c licenseControlModel) map[string]interface{} {
	m := baseControlToMap(c.baseControlModel)
	if !c.LicenseID.IsNull() && c.LicenseID.ValueString() != "" {
		m["license_id"] = c.LicenseID.ValueString()
	}
	if !c.LicenseCategory.IsNull() && c.LicenseCategory.ValueString() != "" {
		m["license_category"] = c.LicenseCategory.ValueString()
	}
	if !c.IsOsiApproved.IsNull() {
		m["is_osi_approved"] = c.IsOsiApproved.ValueBool()
	}
	if !c.IsDeprecated.IsNull() {
		m["is_deprecated"] = c.IsDeprecated.ValueBool()
	}
	if !c.IsFsfLibre.IsNull() {
		m["is_fsf_libre"] = c.IsFsfLibre.ValueBool()
	}
	if !c.Url.IsNull() && c.Url.ValueString() != "" {
		m["url"] = c.Url.ValueString()
	}
	if len(c.AdditionalInfo) > 0 {
		m["additional_info"] = stringSliceFromTypes(c.AdditionalInfo)
	}
	return m
}

func scmControlToMap(c scmControlModel) map[string]interface{} {
	m := map[string]interface{}{
		"id":       c.ID.ValueString(),
		"priority": c.Priority.ValueString(),
		"disabled": c.Disabled.ValueBool(),
	}
	if !c.Scm.IsNull() && c.Scm.ValueString() != "" {
		m["scm"] = c.Scm.ValueString()
	}
	if !c.Entity.IsNull() && c.Entity.ValueString() != "" {
		m["entity"] = c.Entity.ValueString()
	}
	if len(c.Threat) > 0 {
		m["threat"] = stringSliceFromTypes(c.Threat)
	}
	return m
}

func controlsBlockToMaps(block *controlsBlockModel) []map[string]interface{} {
	if block == nil {
		return nil
	}
	result := make([]map[string]interface{}, 0, len(block.Controls))
	for _, c := range block.Controls {
		result = append(result, baseControlToMap(c))
	}
	return result
}

func containerScopeToMaps(block *containerScopeBlockModel) []map[string]interface{} {
	if block == nil {
		return nil
	}
	result := make([]map[string]interface{}, 0, len(block.Controls))
	for _, c := range block.Controls {
		result = append(result, containerControlToMap(c))
	}
	return result
}

func scopeControlsWrapper(controls []map[string]interface{}) map[string]interface{} {
	if controls == nil {
		return map[string]interface{}{"controls": []interface{}{}}
	}
	return map[string]interface{}{"controls": controls}
}

func iacControlsToMaps(block *iacBlockModel) []map[string]interface{} {
	controls := make([]map[string]interface{}, 0, len(block.Controls))
	for _, c := range block.Controls {
		controls = append(controls, iacControlToMap(c))
	}
	return controls
}

func sastControlsToMaps(block *sastBlockModel) []map[string]interface{} {
	controls := make([]map[string]interface{}, 0, len(block.Controls))
	for _, c := range block.Controls {
		controls = append(controls, sastControlToMap(c))
	}
	return controls
}

func licenseControlsToMaps(items []licenseControlModel) []map[string]interface{} {
	controls := make([]map[string]interface{}, 0, len(items))
	for _, c := range items {
		controls = append(controls, licenseControlToMap(c))
	}
	return controls
}

func scmControlsToMaps(items []scmControlModel) []map[string]interface{} {
	controls := make([]map[string]interface{}, 0, len(items))
	for _, c := range items {
		controls = append(controls, scmControlToMap(c))
	}
	return controls
}

// buildContainerImageData populates the container_image policy_data scopes and
// returns the flat list of all controls across every feature scope.
func buildContainerImageData(block *containerImageBlockModel, policy *api_client.ShiftLeftPolicy, policyData map[string]interface{}) []map[string]interface{} {
	policy.FeatureScope = stringSliceFromTypes(block.FeatureScope)
	policyData["feature_scope"] = policy.FeatureScope

	scopes := []struct {
		key      string
		controls []map[string]interface{}
	}{
		{"vulnerabilities", containerScopeToMaps(block.Vulnerabilities)},
		{"secret_detection", containerScopeToMaps(block.SecretDetection)},
		{"container_image_best_practices", containerScopeToMaps(block.ContainerImageBestPractices)},
		{"custom", containerScopeToMaps(block.Custom)},
	}

	var controls []map[string]interface{}
	for _, s := range scopes {
		if len(s.controls) > 0 || containsString(policy.FeatureScope, s.key) {
			policyData[s.key] = scopeControlsWrapper(s.controls)
		}
		controls = append(controls, s.controls...)
	}
	return controls
}

// buildScmScope encodes the scm_posture scope and returns the encoded scope plus its controls.
func buildScmScope(block *scmPostureBlockModel) (json.RawMessage, []map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	scope := map[string][]string{}
	for _, entry := range block.Scope {
		key := entry.Key.ValueString()
		ids := stringSliceFromTypes(entry.Ids)
		if key != "" && len(ids) > 0 {
			scope[key] = ids
		}
	}
	if len(scope) == 0 {
		diags.AddError(
			"Missing SCM scope",
			"scm_posture policies require at least one scope block with a key and ids.",
		)
		return nil, nil, diags
	}
	scopeRaw, err := json.Marshal(scope)
	if err != nil {
		diags.AddError("Failed to encode SCM scope", err.Error())
		return nil, nil, diags
	}
	return scopeRaw, scmControlsToMaps(block.Controls), diags
}
