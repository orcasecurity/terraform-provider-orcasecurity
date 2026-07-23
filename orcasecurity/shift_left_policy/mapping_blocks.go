package shift_left_policy

import (
	"encoding/json"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

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

func rawScopeControls(data map[string]interface{}, key string) []map[string]interface{} {
	section, ok := data[key].(map[string]interface{})
	if !ok {
		return nil
	}
	items, ok := section["controls"].([]interface{})
	if !ok {
		return nil
	}
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, m)
		}
	}
	return result
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
