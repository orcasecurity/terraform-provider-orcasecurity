package api_client

import (
	"encoding/json"
	"fmt"
	"sort"
)

// catalogControlIndex maps control ID to the full catalog control definition.
type catalogControlIndex map[string]map[string]interface{}

// indexControl adds a single control to the index when it has a non-empty id.
func (index catalogControlIndex) indexControl(item interface{}) {
	control, ok := item.(map[string]interface{})
	if !ok {
		return
	}
	if id, ok := control["id"].(string); ok && id != "" {
		index[id] = control
	}
}

// indexControlList adds every control in a slice to the index.
func (index catalogControlIndex) indexControlList(items []interface{}) {
	for _, item := range items {
		index.indexControl(item)
	}
}

// indexFromArray treats the catalog body as a top-level array of controls.
// Returns true when the body parsed as an array and produced at least one control.
func (index catalogControlIndex) indexFromArray(catalogRaw json.RawMessage) bool {
	var asArray []interface{}
	if err := json.Unmarshal(catalogRaw, &asArray); err != nil {
		return false
	}
	index.indexControlList(asArray)
	return len(index) > 0
}

// indexWalk recursively descends the catalog, indexing any "controls" arrays it finds.
func (index catalogControlIndex) indexWalk(node interface{}) {
	switch v := node.(type) {
	case map[string]interface{}:
		if controls, ok := v["controls"].([]interface{}); ok {
			index.indexControlList(controls)
		}
		for _, child := range v {
			index.indexWalk(child)
		}
	case []interface{}:
		for _, item := range v {
			index.indexWalk(item)
		}
	}
}

func buildCatalogControlIndex(catalogRaw json.RawMessage, policyType string) (catalogControlIndex, error) {
	index := catalogControlIndex{}
	if len(catalogRaw) == 0 {
		return index, nil
	}

	if index.indexFromArray(catalogRaw) {
		return index, nil
	}

	var catalog map[string]interface{}
	if err := json.Unmarshal(catalogRaw, &catalog); err != nil {
		return nil, err
	}

	index.indexWalk(catalog)

	// Some policy types return controls at the top level as an array.
	if controls, ok := catalog["controls"].([]interface{}); ok {
		index.indexControlList(controls)
	}

	return index, nil
}

// findByTitle returns the catalog control whose title matches the given value (exact match).
func (index catalogControlIndex) findByTitle(title interface{}) (map[string]interface{}, bool) {
	wanted, ok := title.(string)
	if !ok || wanted == "" {
		return nil, false
	}
	for _, control := range index {
		if controlTitle, ok := control["title"].(string); ok && controlTitle == wanted {
			return control, true
		}
	}
	return nil, false
}

func cloneMap(src map[string]interface{}) map[string]interface{} {
	raw, _ := json.Marshal(src)
	dst := map[string]interface{}{}
	_ = json.Unmarshal(raw, &dst)
	return dst
}

// mergeConditions deep-merges override conditions onto the merged map in place.
func mergeConditions(merged map[string]interface{}, value interface{}) {
	overrideConditions, ok := value.(map[string]interface{})
	if !ok || len(overrideConditions) == 0 {
		return
	}
	if baseConditions, ok := merged["conditions"].(map[string]interface{}); ok {
		merged["conditions"] = mergeControlMaps(baseConditions, overrideConditions)
	} else {
		merged["conditions"] = overrideConditions
	}
}

func mergeControlMaps(base, override map[string]interface{}) map[string]interface{} {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	merged := cloneMap(base)
	for key, value := range override {
		if key == "conditions" {
			mergeConditions(merged, value)
			continue
		}
		if value != nil {
			merged[key] = value
		}
	}
	return merged
}

func enrichControlsSlice(controls []interface{}, index catalogControlIndex) ([]interface{}, error) {
	if len(controls) == 0 {
		return controls, nil
	}

	enriched := make([]interface{}, 0, len(controls))
	for _, item := range controls {
		override, ok := item.(map[string]interface{})
		if !ok {
			enriched = append(enriched, item)
			continue
		}

		id, _ := override["id"].(string)
		if id == "" {
			// No catalog id: try to resolve it from the catalog by title so callers
			// can reference a catalog control without a data source lookup.
			if base, found := index.findByTitle(override["title"]); found {
				enriched = append(enriched, mergeControlMaps(base, override))
				continue
			}
			// Otherwise this is a brand-new custom control: keep it as-is.
			enriched = append(enriched, override)
			continue
		}

		base, found := index[id]
		if !found {
			return nil, fmt.Errorf("unknown control id %q (not found in catalog)", id)
		}

		enriched = append(enriched, mergeControlMaps(base, override))
	}
	return enriched, nil
}

// enrichNestedMap enriches a nested object: its "controls" slice (if any) plus any deeper maps.
func enrichNestedMap(typed map[string]interface{}, index catalogControlIndex) error {
	if controls, ok := typed["controls"].([]interface{}); ok {
		enriched, err := enrichControlsSlice(controls, index)
		if err != nil {
			return err
		}
		typed["controls"] = enriched
	}
	return enrichControlsInMap(typed, index)
}

func enrichControlsInMap(data map[string]interface{}, index catalogControlIndex) error {
	for key, value := range data {
		switch typed := value.(type) {
		case map[string]interface{}:
			if err := enrichNestedMap(typed, index); err != nil {
				return err
			}
			data[key] = typed
		case []interface{}:
			if key != "controls" {
				continue
			}
			enriched, err := enrichControlsSlice(typed, index)
			if err != nil {
				return err
			}
			data[key] = enriched
		}
	}
	return nil
}

// EnrichShiftLeftPolicyFromCatalog fills missing control fields from the policy type catalog.
func (client *APIClient) EnrichShiftLeftPolicyFromCatalog(policyType string, policy *ShiftLeftPolicy) error {
	catalog, err := client.GetShiftLeftPolicyCatalogControls(policyType)
	if err != nil {
		return err
	}

	index, err := buildCatalogControlIndex(catalog.Body, policyType)
	if err != nil {
		return err
	}
	if len(index) == 0 {
		return fmt.Errorf("catalog for policy type %q returned no controls", policyType)
	}

	if policy.Controls, err = enrichControlsRaw(policy.Controls, index); err != nil {
		return err
	}
	if policy.PolicyData, err = enrichPolicyDataRaw(policy.PolicyData, index); err != nil {
		return err
	}
	return nil
}

// enrichControlsRaw enriches a raw JSON array of controls. Returns the input unchanged when empty.
func enrichControlsRaw(raw json.RawMessage, index catalogControlIndex) (json.RawMessage, error) {
	if len(raw) == 0 {
		return raw, nil
	}
	var controls []interface{}
	if err := json.Unmarshal(raw, &controls); err != nil {
		return nil, err
	}
	enriched, err := enrichControlsSlice(controls, index)
	if err != nil {
		return nil, err
	}
	return json.Marshal(enriched)
}

// enrichPolicyDataRaw enriches the controls nested anywhere inside a raw policy_data object.
func enrichPolicyDataRaw(raw json.RawMessage, index catalogControlIndex) (json.RawMessage, error) {
	if len(raw) == 0 {
		return raw, nil
	}
	var policyData map[string]interface{}
	if err := json.Unmarshal(raw, &policyData); err != nil {
		return nil, err
	}
	if err := enrichControlsInMap(policyData, index); err != nil {
		return nil, err
	}
	return json.Marshal(policyData)
}

func toControlMaps(items []interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	return out
}

// catalogControlsByScope groups catalog controls by feature scope. Container-style
// catalogs key controls by top-level scope name; flat catalogs use the "" key.
func catalogControlsByScope(catalogRaw json.RawMessage) map[string][]map[string]interface{} {
	result := map[string][]map[string]interface{}{}
	if len(catalogRaw) == 0 {
		return result
	}

	var asArray []interface{}
	if err := json.Unmarshal(catalogRaw, &asArray); err == nil && len(asArray) > 0 {
		result[""] = toControlMaps(asArray)
		return result
	}

	var catalog map[string]interface{}
	if err := json.Unmarshal(catalogRaw, &catalog); err != nil {
		return result
	}
	if controls, ok := catalog["controls"].([]interface{}); ok {
		result[""] = toControlMaps(controls)
	}
	for scope, value := range catalog {
		obj, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		if controls, ok := obj["controls"].([]interface{}); ok {
			result[scope] = toControlMaps(controls)
		}
	}
	return result
}

// catalogEntry reduces a catalog control to the minimal fields needed in a policy.
// The remaining fields (title, conditions, ...) are added later by enrichment.
func catalogEntry(control map[string]interface{}) (map[string]interface{}, bool) {
	id, ok := control["id"].(string)
	if !ok || id == "" {
		return nil, false
	}
	entry := map[string]interface{}{"id": id, "disabled": false}
	if priority, ok := control["priority"].(string); ok && priority != "" {
		entry["priority"] = priority
	} else {
		entry["priority"] = "MEDIUM"
	}
	return entry, true
}

// unionPolicyDataControls returns every control referenced anywhere in policy_data.
func unionPolicyDataControls(policyData map[string]interface{}) []interface{} {
	if controls, ok := policyData["controls"].([]interface{}); ok {
		return controls
	}
	var union []interface{}
	for _, value := range policyData {
		obj, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		if controls, ok := obj["controls"].([]interface{}); ok {
			union = append(union, controls...)
		}
	}
	return union
}

// scopeEntries builds the minimal control entries for a single catalog scope.
func scopeEntries(controls []map[string]interface{}) []interface{} {
	entries := make([]interface{}, 0, len(controls))
	for _, control := range controls {
		if entry, ok := catalogEntry(control); ok {
			entries = append(entries, entry)
		}
	}
	return entries
}

// injectScopeControls writes the catalog entries for a scope into policy_data.
// Scope "" stores controls at the top level; named scopes nest under their key.
func injectScopeControls(policyData map[string]interface{}, scopeKey string, entries []interface{}) {
	if scopeKey == "" {
		policyData["controls"] = entries
		return
	}
	policyData[scopeKey] = map[string]interface{}{"controls": entries}
}

// AddAllCatalogControls injects every catalog control for the given scopes into the
// policy. Use scope "" for non-grouped policy types (controls live at the top level);
// for container_image pass the feature scope names (e.g. "vulnerabilities").
func (client *APIClient) AddAllCatalogControls(policyType string, policy *ShiftLeftPolicy, scopeKeys []string) error {
	if len(scopeKeys) == 0 {
		return nil
	}

	catalog, err := client.GetShiftLeftPolicyCatalogControls(policyType)
	if err != nil {
		return err
	}
	byScope := catalogControlsByScope(catalog.Body)

	policyData := map[string]interface{}{}
	if len(policy.PolicyData) > 0 {
		if err := json.Unmarshal(policy.PolicyData, &policyData); err != nil {
			return err
		}
	}

	for _, scopeKey := range scopeKeys {
		injectScopeControls(policyData, scopeKey, scopeEntries(byScope[scopeKey]))
	}

	pdRaw, err := json.Marshal(policyData)
	if err != nil {
		return err
	}
	policy.PolicyData = pdRaw

	ctrlRaw, err := json.Marshal(unionPolicyDataControls(policyData))
	if err != nil {
		return err
	}
	policy.Controls = ctrlRaw
	return nil
}

// CatalogControlSummary is a flattened catalog control entry.
type CatalogControlSummary struct {
	ID       string
	Title    string
	Category string
	Priority string
}

// FlattenCatalogControls extracts control summaries from a nested catalog response.
func FlattenCatalogControls(catalogRaw json.RawMessage) []CatalogControlSummary {
	index, _ := buildCatalogControlIndex(catalogRaw, "")
	controls := make([]CatalogControlSummary, 0, len(index))
	for id, control := range index {
		c := CatalogControlSummary{ID: id}
		if title, ok := control["title"].(string); ok {
			c.Title = title
		}
		if category, ok := control["category"].(string); ok {
			c.Category = category
		}
		if priority, ok := control["priority"].(string); ok {
			c.Priority = priority
		}
		controls = append(controls, c)
	}
	sort.Slice(controls, func(i, j int) bool {
		return controls[i].ID < controls[j].ID
	})
	return controls
}
