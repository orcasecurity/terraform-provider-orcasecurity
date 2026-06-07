package api_client

import (
	"encoding/json"
	"fmt"
	"sort"
)

// catalogControlIndex maps control ID to the full catalog control definition.
type catalogControlIndex map[string]map[string]interface{}

func buildCatalogControlIndex(catalogRaw json.RawMessage, policyType string) (catalogControlIndex, error) {
	index := catalogControlIndex{}
	if len(catalogRaw) == 0 {
		return index, nil
	}

	var asArray []interface{}
	if err := json.Unmarshal(catalogRaw, &asArray); err == nil {
		for _, item := range asArray {
			if control, ok := item.(map[string]interface{}); ok {
				if id, ok := control["id"].(string); ok && id != "" {
					index[id] = control
				}
			}
		}
		if len(index) > 0 {
			return index, nil
		}
	}

	var catalog map[string]interface{}
	if err := json.Unmarshal(catalogRaw, &catalog); err != nil {
		return nil, err
	}

	var walk func(node interface{})
	walk = func(node interface{}) {
		switch v := node.(type) {
		case map[string]interface{}:
			if controls, ok := v["controls"].([]interface{}); ok {
				for _, item := range controls {
					if control, ok := item.(map[string]interface{}); ok {
						if id, ok := control["id"].(string); ok && id != "" {
							index[id] = control
						}
					}
				}
			}
			for _, child := range v {
				walk(child)
			}
		case []interface{}:
			for _, item := range v {
				walk(item)
			}
		}
	}

	walk(catalog)

	// Some policy types return controls at the top level as an array.
	if controls, ok := catalog["controls"].([]interface{}); ok {
		for _, item := range controls {
			if control, ok := item.(map[string]interface{}); ok {
				if id, ok := control["id"].(string); ok && id != "" {
					index[id] = control
				}
			}
		}
	}

	return index, nil
}

func cloneMap(src map[string]interface{}) map[string]interface{} {
	raw, _ := json.Marshal(src)
	dst := map[string]interface{}{}
	_ = json.Unmarshal(raw, &dst)
	return dst
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
		switch key {
		case "priority", "disabled", "title":
			if value != nil {
				merged[key] = value
			}
		case "conditions":
			if overrideConditions, ok := value.(map[string]interface{}); ok && len(overrideConditions) > 0 {
				if baseConditions, ok := merged["conditions"].(map[string]interface{}); ok {
					merged["conditions"] = mergeControlMaps(baseConditions, overrideConditions)
				} else {
					merged["conditions"] = overrideConditions
				}
			}
		default:
			if value != nil {
				merged[key] = value
			}
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
			return nil, fmt.Errorf("control id is required")
		}

		base, found := index[id]
		if !found {
			return nil, fmt.Errorf("unknown control id %q (not found in catalog)", id)
		}

		enriched = append(enriched, mergeControlMaps(base, override))
	}
	return enriched, nil
}

func enrichControlsInMap(data map[string]interface{}, index catalogControlIndex) error {
	for key, value := range data {
		switch typed := value.(type) {
		case map[string]interface{}:
			if controls, ok := typed["controls"].([]interface{}); ok {
				enriched, err := enrichControlsSlice(controls, index)
				if err != nil {
					return err
				}
				typed["controls"] = enriched
			}
			if err := enrichControlsInMap(typed, index); err != nil {
				return err
			}
			data[key] = typed
		case []interface{}:
			if key == "controls" {
				enriched, err := enrichControlsSlice(typed, index)
				if err != nil {
					return err
				}
				data[key] = enriched
			}
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

	if len(policy.Controls) > 0 {
		var controls []interface{}
		if err := json.Unmarshal(policy.Controls, &controls); err != nil {
			return err
		}
		enriched, err := enrichControlsSlice(controls, index)
		if err != nil {
			return err
		}
		policy.Controls, err = json.Marshal(enriched)
		if err != nil {
			return err
		}
	}

	if len(policy.PolicyData) > 0 {
		var policyData map[string]interface{}
		if err := json.Unmarshal(policy.PolicyData, &policyData); err != nil {
			return err
		}
		if err := enrichControlsInMap(policyData, index); err != nil {
			return err
		}
		policy.PolicyData, err = json.Marshal(policyData)
		if err != nil {
			return err
		}
	}

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
