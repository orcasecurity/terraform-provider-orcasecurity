package shift_left_policy

import (
	"encoding/json"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringSliceFromTypes(values []types.String) []string {
	result := make([]string, 0, len(values))
	for _, v := range values {
		if !v.IsNull() && !v.IsUnknown() {
			result = append(result, v.ValueString())
		}
	}
	return result
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

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
		"id":       c.ID.ValueString(),
		"priority": c.Priority.ValueString(),
		"disabled": c.Disabled.ValueBool(),
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

func validateTypeBlock(policyType string, model *shiftLeftPolicyResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	hasBlock := func(set bool, name string) {
		if !set {
			diags.AddError("Missing type configuration block", fmt.Sprintf("Policy type %q requires the %q block to be set.", policyType, name))
		}
	}

	switch policyType {
	case "iac":
		hasBlock(model.Iac != nil, "iac")
	case "sast":
		hasBlock(model.Sast != nil, "sast")
	case "file_system":
		hasBlock(model.FileSystem != nil, "file_system")
	case "file_system_vulnerabilities":
		hasBlock(model.FileSystemVulnerabilities != nil, "file_system_vulnerabilities")
	case "file_system_secret_detection":
		hasBlock(model.FileSystemSecretDetection != nil, "file_system_secret_detection")
	case "container_image":
		hasBlock(model.ContainerImage != nil, "container_image")
	case "scm_posture":
		hasBlock(model.ScmPosture != nil, "scm_posture")
	case "licenses":
		hasBlock(model.Licenses != nil, "licenses")
	case "sca":
		hasBlock(model.Sca != nil, "sca")
	default:
		diags.AddError("Unsupported policy type", fmt.Sprintf("Unknown policy type %q.", policyType))
	}
	return diags
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

// buildControlsAndData dispatches per policy type, returning the controls slice and policy_data map.
func buildControlsAndData(model *shiftLeftPolicyResourceModel, policy *api_client.ShiftLeftPolicy) ([]map[string]interface{}, map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	policyData := map[string]interface{}{}
	var controls []map[string]interface{}

	switch model.Type.ValueString() {
	case "iac":
		controls = iacControlsToMaps(model.Iac)
		policyData["controls"] = controls
	case "sast":
		controls = sastControlsToMaps(model.Sast)
		policyData["controls"] = controls
	case "file_system":
		controls = controlsBlockToMaps(model.FileSystem)
		policyData["controls"] = controls
	case "file_system_vulnerabilities":
		controls = controlsBlockToMaps(model.FileSystemVulnerabilities)
		policyData["controls"] = controls
	case "file_system_secret_detection":
		controls = controlsBlockToMaps(model.FileSystemSecretDetection)
		policyData["controls"] = controls
	case "container_image":
		controls = buildContainerImageData(model.ContainerImage, policy, policyData)
	case "scm_posture":
		scopeRaw, scmControls, d := buildScmScope(model.ScmPosture)
		diags.Append(d...)
		if diags.HasError() {
			return nil, nil, diags
		}
		policy.Scope = scopeRaw
		controls = scmControls
		policyData["controls"] = controls
	case "licenses":
		controls = licenseControlsToMaps(model.Licenses.Controls)
		policyData["controls"] = controls
	case "sca":
		controls = licenseControlsToMaps(model.Sca.Controls)
		policyData["controls"] = controls
	}

	return controls, policyData, diags
}

func encodeJSONField(value interface{}, label string, diags *diag.Diagnostics) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		diags.AddError("Failed to encode "+label, err.Error())
		return nil
	}
	return raw
}

func planToAPI(model *shiftLeftPolicyResourceModel) (api_client.ShiftLeftPolicy, diag.Diagnostics) {
	var diags diag.Diagnostics
	policyType := model.Type.ValueString()
	diags.Append(validateTypeBlock(policyType, model)...)
	if diags.HasError() {
		return api_client.ShiftLeftPolicy{}, diags
	}

	policy := api_client.ShiftLeftPolicy{
		Name:                     model.Name.ValueString(),
		Description:              model.Description.ValueString(),
		Disabled:                 model.Disabled.ValueBool(),
		WarnMode:                 model.WarnMode.ValueBool(),
		PriorityFailureThreshold: model.PriorityFailureThreshold.ValueString(),
		Type:                     policyType,
		ProjectsIds:              stringSliceFromTypes(model.ProjectsIds),
	}

	controls, policyData, d := buildControlsAndData(model, &policy)
	diags.Append(d...)
	if diags.HasError() {
		return api_client.ShiftLeftPolicy{}, diags
	}

	if len(controls) > 0 {
		if policy.Controls = encodeJSONField(controls, "controls", &diags); diags.HasError() {
			return api_client.ShiftLeftPolicy{}, diags
		}
	}
	if len(policyData) > 0 {
		if policy.PolicyData = encodeJSONField(policyData, "policy_data", &diags); diags.HasError() {
			return api_client.ShiftLeftPolicy{}, diags
		}
	}

	return policy, diags
}

func mapSeveritiesToConditions(m map[string]interface{}, c *conditionsModel) {
	sev, ok := m["severities"].(map[string]interface{})
	if !ok {
		return
	}
	if op, ok := sev["operator"].(string); ok {
		c.SeveritiesOperator = types.StringValue(op)
	}
	vals, ok := sev["values"].([]interface{})
	if !ok {
		return
	}
	for _, val := range vals {
		if str, ok := val.(string); ok {
			c.SeveritiesValues = append(c.SeveritiesValues, types.StringValue(str))
		}
	}
}

func mapToConditions(m map[string]interface{}) *conditionsModel {
	if m == nil {
		return nil
	}
	c := &conditionsModel{}
	if v, ok := m["fix_available"].(bool); ok {
		c.FixAvailable = types.BoolValue(v)
	}
	if v, ok := m["from_base_image"].(bool); ok {
		c.FromBaseImage = types.BoolValue(v)
	}
	if v, ok := m["days_from_discovery"].(float64); ok {
		c.DaysFromDiscovery = types.Int64Value(int64(v))
	}
	if v, ok := m["days_from_fix"].(float64); ok {
		c.DaysFromFix = types.Int64Value(int64(v))
	}
	if v, ok := m["has_exploit"].(bool); ok {
		c.HasExploit = types.BoolValue(v)
	}
	mapSeveritiesToConditions(m, c)
	return c
}

func mapToBaseControl(m map[string]interface{}) baseControlModel {
	c := baseControlModel{}
	if v, ok := m["id"].(string); ok {
		c.ID = types.StringValue(v)
	}
	if v, ok := m["title"].(string); ok {
		c.Title = types.StringValue(v)
	}
	if v, ok := m["priority"].(string); ok {
		c.Priority = types.StringValue(v)
	}
	if v, ok := m["disabled"].(bool); ok {
		c.Disabled = types.BoolValue(v)
	}
	if cond, ok := m["conditions"].(map[string]interface{}); ok {
		c.Conditions = mapToConditions(cond)
	}
	return c
}

func stringSliceToTypes(values []string) []types.String {
	if len(values) == 0 {
		return nil
	}
	result := make([]types.String, len(values))
	for i, v := range values {
		result[i] = types.StringValue(v)
	}
	return result
}

func controlIDFromTopLevel(control map[string]interface{}, topLevel []map[string]interface{}) string {
	if id, ok := control["id"].(string); ok && id != "" {
		return id
	}
	title, _ := control["title"].(string)
	if title == "" {
		return ""
	}
	for _, candidate := range topLevel {
		if candidateTitle, ok := candidate["title"].(string); ok && candidateTitle == title {
			if id, ok := candidate["id"].(string); ok && id != "" {
				return id
			}
		}
	}
	return ""
}

func isStringSet(value types.String) bool {
	return !value.IsNull() && !value.IsUnknown() && value.ValueString() != ""
}

func mergeBaseControlFromPlan(dst *baseControlModel, src baseControlModel) {
	if isStringSet(src.ID) {
		dst.ID = src.ID
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
	for i := range dst.Controls {
		if i >= len(src.Controls) {
			continue
		}
		mergeBaseControlFromPlan(&dst.Controls[i].baseControlModel, src.Controls[i].baseControlModel)
		mergeLicenseExtrasFromPlan(&dst.Controls[i], src.Controls[i])
	}
}

func mergeProjectsIdsFromPlan(state, plan *shiftLeftPolicyResourceModel) {
	if len(plan.ProjectsIds) == 0 {
		state.ProjectsIds = nil
	} else if len(state.ProjectsIds) == 0 {
		state.ProjectsIds = plan.ProjectsIds
	}
}

func mergeStateFromPlan(state, plan *shiftLeftPolicyResourceModel) {
	mergeProjectsIdsFromPlan(state, plan)

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

func controlsFromRaw(raw json.RawMessage) []map[string]interface{} {
	if len(raw) == 0 {
		return nil
	}
	var controls []map[string]interface{}
	_ = json.Unmarshal(raw, &controls)
	return controls
}

func policyDataFromRaw(raw json.RawMessage) map[string]interface{} {
	if len(raw) == 0 {
		return map[string]interface{}{}
	}
	var data map[string]interface{}
	_ = json.Unmarshal(raw, &data)
	return data
}

func controlsFromPolicyData(data map[string]interface{}) []map[string]interface{} {
	if controlsRaw, ok := data["controls"].([]interface{}); ok {
		result := make([]map[string]interface{}, 0, len(controlsRaw))
		for _, item := range controlsRaw {
			if m, ok := item.(map[string]interface{}); ok {
				result = append(result, m)
			}
		}
		return result
	}
	return nil
}

func scopeControlsFromPolicyData(data map[string]interface{}, key string, topLevelControls []map[string]interface{}) *containerScopeBlockModel {
	section, ok := data[key].(map[string]interface{})
	if !ok {
		return nil
	}
	controlsRaw, ok := section["controls"].([]interface{})
	if !ok {
		return &containerScopeBlockModel{}
	}
	block := &containerScopeBlockModel{}
	for _, item := range controlsRaw {
		if m, ok := item.(map[string]interface{}); ok {
			if id := controlIDFromTopLevel(m, topLevelControls); id != "" {
				m["id"] = id
			}
			base := mapToBaseControl(m)
			ctrl := containerControlModel{baseControlModel: base}
			if v, ok := m["origin"].(string); ok {
				ctrl.Origin = types.StringValue(v)
			}
			block.Controls = append(block.Controls, ctrl)
		}
	}
	return block
}

func stringListFromMap(m map[string]interface{}, key string) []types.String {
	vals, ok := m[key].([]interface{})
	if !ok {
		return nil
	}
	result := make([]types.String, 0, len(vals))
	for _, v := range vals {
		if s, ok := v.(string); ok {
			result = append(result, types.StringValue(s))
		}
	}
	return result
}

func mapToIacControl(m map[string]interface{}) iacControlModel {
	ctrl := iacControlModel{baseControlModel: mapToBaseControl(m)}
	ctrl.Frameworks = stringListFromMap(m, "frameworks")
	if v, ok := m["orca_alert_rule_type"].(string); ok {
		ctrl.OrcaAlertRuleType = types.StringValue(v)
	}
	return ctrl
}

func buildIacBlock(controls []map[string]interface{}) *iacBlockModel {
	block := &iacBlockModel{}
	for _, m := range controls {
		block.Controls = append(block.Controls, mapToIacControl(m))
	}
	return block
}

func mapToSastControl(m map[string]interface{}) sastControlModel {
	ctrl := sastControlModel{baseControlModel: mapToBaseControl(m)}
	ctrl.Languages = stringListFromMap(m, "languages")
	ctrl.Owasp = stringListFromMap(m, "owasp")
	ctrl.Cwe = stringListFromMap(m, "cwe")
	if v, ok := m["section"].(string); ok {
		ctrl.Section = types.StringValue(v)
	}
	if v, ok := m["confidence"].(string); ok {
		ctrl.Confidence = types.StringValue(v)
	}
	if v, ok := m["impact"].(string); ok {
		ctrl.Impact = types.StringValue(v)
	}
	if v, ok := m["likelihood"].(string); ok {
		ctrl.Likelihood = types.StringValue(v)
	}
	return ctrl
}

func buildSastBlock(controls []map[string]interface{}) *sastBlockModel {
	block := &sastBlockModel{}
	for _, m := range controls {
		block.Controls = append(block.Controls, mapToSastControl(m))
	}
	return block
}

func buildControlsBlock(controls []map[string]interface{}) *controlsBlockModel {
	block := &controlsBlockModel{}
	for _, m := range controls {
		block.Controls = append(block.Controls, mapToBaseControl(m))
	}
	return block
}

func buildContainerImageBlock(apiPolicy *api_client.ShiftLeftPolicy, policyData map[string]interface{}, controls []map[string]interface{}) *containerImageBlockModel {
	block := &containerImageBlockModel{
		FeatureScope:                stringSliceToTypes(apiPolicy.FeatureScope),
		Vulnerabilities:             scopeControlsFromPolicyData(policyData, "vulnerabilities", controls),
		SecretDetection:             scopeControlsFromPolicyData(policyData, "secret_detection", controls),
		ContainerImageBestPractices: scopeControlsFromPolicyData(policyData, "container_image_best_practices", controls),
		Custom:                      scopeControlsFromPolicyData(policyData, "custom", controls),
	}
	if len(block.FeatureScope) == 0 {
		block.FeatureScope = stringListFromMap(policyData, "feature_scope")
	}
	return block
}

func mapToScmControl(m map[string]interface{}) scmControlModel {
	ctrl := scmControlModel{}
	if v, ok := m["id"].(string); ok {
		ctrl.ID = types.StringValue(v)
	}
	if v, ok := m["priority"].(string); ok {
		ctrl.Priority = types.StringValue(v)
	}
	if v, ok := m["disabled"].(bool); ok {
		ctrl.Disabled = types.BoolValue(v)
	}
	if v, ok := m["scm"].(string); ok {
		ctrl.Scm = types.StringValue(v)
	}
	if v, ok := m["entity"].(string); ok {
		ctrl.Entity = types.StringValue(v)
	}
	ctrl.Threat = stringListFromMap(m, "threat")
	return ctrl
}

func buildScmPostureBlock(apiPolicy *api_client.ShiftLeftPolicy, controls []map[string]interface{}) *scmPostureBlockModel {
	block := &scmPostureBlockModel{}
	if len(apiPolicy.Scope) > 0 {
		var scope map[string][]string
		_ = json.Unmarshal(apiPolicy.Scope, &scope)
		for key, ids := range scope {
			block.Scope = append(block.Scope, scmScopeEntryModel{
				Key: types.StringValue(key),
				Ids: stringSliceToTypes(ids),
			})
		}
	}
	for _, m := range controls {
		block.Controls = append(block.Controls, mapToScmControl(m))
	}
	return block
}

func buildLicensesBlock(controls []map[string]interface{}) *licensesBlockModel {
	block := &licensesBlockModel{}
	for _, m := range controls {
		block.Controls = append(block.Controls, mapToLicenseControl(m))
	}
	return block
}

func resolveControls(apiPolicy *api_client.ShiftLeftPolicy, policyData map[string]interface{}) []map[string]interface{} {
	controls := controlsFromRaw(apiPolicy.Controls)
	if len(controls) == 0 {
		controls = controlsFromPolicyData(policyData)
	}
	return controls
}

func applyTypeBlockToState(model *shiftLeftPolicyResourceModel, policyType string, apiPolicy *api_client.ShiftLeftPolicy, policyData map[string]interface{}, controls []map[string]interface{}) {
	switch policyType {
	case "iac":
		model.Iac = buildIacBlock(controls)
	case "sast":
		model.Sast = buildSastBlock(controls)
	case "file_system":
		model.FileSystem = buildControlsBlock(controls)
	case "file_system_vulnerabilities":
		model.FileSystemVulnerabilities = buildControlsBlock(controls)
	case "file_system_secret_detection":
		model.FileSystemSecretDetection = buildControlsBlock(controls)
	case "container_image":
		model.ContainerImage = buildContainerImageBlock(apiPolicy, policyData, controls)
	case "scm_posture":
		model.ScmPosture = buildScmPostureBlock(apiPolicy, controls)
	case "licenses":
		model.Licenses = buildLicensesBlock(controls)
	case "sca":
		model.Sca = buildLicensesBlock(controls)
	}
}

func apiToState(apiPolicy *api_client.ShiftLeftPolicy, existing *shiftLeftPolicyResourceModel) *shiftLeftPolicyResourceModel {
	model := &shiftLeftPolicyResourceModel{
		ID:                       types.StringValue(apiPolicy.ID),
		Type:                     types.StringValue(apiPolicy.Type),
		Name:                     types.StringValue(apiPolicy.Name),
		Description:              types.StringValue(apiPolicy.Description),
		Disabled:                 types.BoolValue(apiPolicy.Disabled),
		WarnMode:                 types.BoolValue(apiPolicy.WarnMode),
		PriorityFailureThreshold: types.StringValue(apiPolicy.PriorityFailureThreshold),
		ProjectsIds:              stringSliceToTypes(apiPolicy.ProjectsIds),
		Builtin:                  types.BoolValue(apiPolicy.Builtin),
	}

	policyType := apiPolicy.Type
	if policyType == "" && existing != nil {
		policyType = existing.Type.ValueString()
		model.Type = types.StringValue(policyType)
	}

	policyData := policyDataFromRaw(apiPolicy.PolicyData)
	controls := resolveControls(apiPolicy, policyData)

	applyTypeBlockToState(model, policyType, apiPolicy, policyData, controls)

	if existing != nil {
		mergeStateFromPlan(model, existing)
	}

	return model
}

// stateFromPlanAfterWrite anchors post-create/update state on the applied plan.
// Nested container controls in API responses are not reliably round-tripped immediately after write.
func stateFromPlanAfterWrite(plan *shiftLeftPolicyResourceModel, apiPolicy *api_client.ShiftLeftPolicy) *shiftLeftPolicyResourceModel {
	state := *plan
	state.ID = types.StringValue(apiPolicy.ID)
	state.Builtin = types.BoolValue(apiPolicy.Builtin)
	if len(plan.ProjectsIds) == 0 {
		state.ProjectsIds = nil
	}
	return &state
}

func stringValueFromMap(m map[string]interface{}, key string) types.String {
	if v, ok := m[key].(string); ok {
		return types.StringValue(v)
	}
	return types.String{}
}

func boolValueFromMap(m map[string]interface{}, key string) types.Bool {
	if v, ok := m[key].(bool); ok {
		return types.BoolValue(v)
	}
	return types.Bool{}
}

func mapToLicenseControl(m map[string]interface{}) licenseControlModel {
	return licenseControlModel{
		baseControlModel: mapToBaseControl(m),
		LicenseID:        stringValueFromMap(m, "license_id"),
		LicenseCategory:  stringValueFromMap(m, "license_category"),
		Url:              stringValueFromMap(m, "url"),
		IsOsiApproved:    boolValueFromMap(m, "is_osi_approved"),
		IsDeprecated:     boolValueFromMap(m, "is_deprecated"),
		IsFsfLibre:       boolValueFromMap(m, "is_fsf_libre"),
		AdditionalInfo:   stringListFromMap(m, "additional_info"),
	}
}

func parseImportID(id string) (policyType, policyID string, err error) {
	for i := 0; i < len(id); i++ {
		if id[i] == '/' {
			return id[:i], id[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("import ID must be in the format <type>/<id>, got %q", id)
}
