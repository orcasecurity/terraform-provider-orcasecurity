package business_unit

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Intermediate config variables, in the order the backend's
// convert_legacy_to_intermediate emits them.
const (
	varCloudProviders   = "CloudProviders"
	varCloudVendorIDs   = "CloudVendorIDs"
	varAccountTags      = "AccountTags"
	varInventoryTags    = "InventoryTags"
	varCustomTags       = "CustomTags"
	varAppSecProjectIDs = "AppSecProjectIDs"
)

// configFromModel turns filter_data / shiftleft_filter_data into an intermediate
// config: one `some` leaf per populated attribute, combined with `or` (the union
// semantics legacy filters had). Returns nil when no attribute has values.
func configFromModel(filter *businessUnitFilterModel, shiftLeft *businessUnitShiftLeftFilterModel) (json.RawMessage, error) {
	var leaves []any
	add := func(variable string, values []types.String) {
		if len(values) == 0 {
			return
		}
		leaves = append(leaves, map[string]map[string][]string{"some": {variable: typesSliceToStrings(values)}})
	}

	if filter != nil {
		add(varCloudProviders, filter.CloudProviders)
		add(varCloudVendorIDs, getCloudVendorIds(filter))
		add(varAccountTags, filter.AccountTags)
		add(varInventoryTags, filter.CloudTags)
		add(varCustomTags, filter.CustomTags)
	}
	if shiftLeft != nil {
		add(varAppSecProjectIDs, shiftLeft.ShiftLeftProjects)
	}

	switch len(leaves) {
	case 0:
		return nil, nil
	case 1:
		return json.Marshal(leaves[0])
	default:
		return json.Marshal(map[string][]any{"or": leaves})
	}
}

// modelFromConfig is the inverse of configFromModel. ok is false when the config
// uses anything filter_data / shiftleft_filter_data cannot express: and, all,
// nested branches, or variables without an attribute (HierarchyPath, K8sNamespaces).
func modelFromConfig(config json.RawMessage) (filter *businessUnitFilterModel, shiftLeft *businessUnitShiftLeftFilterModel, ok bool) {
	var root map[string]json.RawMessage
	if len(config) == 0 || string(config) == "null" {
		return nil, nil, true
	}
	if err := json.Unmarshal(config, &root); err != nil {
		return nil, nil, false
	}
	if len(root) == 0 {
		return nil, nil, true
	}

	nodes := []json.RawMessage{config}
	if children, isOr := root["or"]; isOr && len(root) == 1 {
		if err := json.Unmarshal(children, &nodes); err != nil || len(nodes) == 0 {
			return nil, nil, false
		}
	}

	values := map[string][]string{}
	for _, node := range nodes {
		variable, leafValues, isLeaf := parseSomeLeaf(node)
		if !isLeaf {
			return nil, nil, false
		}
		values[variable] = append(values[variable], leafValues...)
	}

	filter = &businessUnitFilterModel{}
	hasFilter := false
	for variable, leafValues := range values {
		converted := stringSliceToTypes(leafValues)
		switch variable {
		case varCloudProviders:
			filter.CloudProviders = converted
		case varCloudVendorIDs:
			filter.CloudAccounts = converted
		case varAccountTags:
			filter.AccountTags = converted
		case varInventoryTags:
			filter.CloudTags = converted
		case varCustomTags:
			filter.CustomTags = converted
		case varAppSecProjectIDs:
			shiftLeft = &businessUnitShiftLeftFilterModel{ShiftLeftProjects: converted}
			continue
		default:
			return nil, nil, false
		}
		hasFilter = true
	}
	if !hasFilter {
		filter = nil
	}
	return filter, shiftLeft, true
}

func parseSomeLeaf(node json.RawMessage) (string, []string, bool) {
	var leaf map[string]map[string][]string
	if err := json.Unmarshal(node, &leaf); err != nil || len(leaf) != 1 {
		return "", nil, false
	}
	spec, isSome := leaf["some"]
	if !isSome || len(spec) != 1 {
		return "", nil, false
	}
	for variable, values := range spec {
		return variable, values, len(values) > 0
	}
	return "", nil, false
}
