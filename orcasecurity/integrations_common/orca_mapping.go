package integrations_common

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// The Orca field-mapping config for template integrations (jira, sn_incidents, monday, …) is a
// JSON object whose values are lists of entries. The common entry is a "pull this Orca alert
// field" reference, which the API represents verbatim as `{"orca": "<field>"}`. Writing that
// wrapper for every field is noisy in HCL, so users may write a bare string (`"<field>"`)
// instead: the provider expands it to `{"orca": "<field>"}` on the way to the API. Plans stay
// stable in the other direction via the OrcaMappingType semantic-equality check (see
// orca_mapping_type.go), which treats the bare string and the `{"orca": ...}` object as equal —
// so there is no collapse-on-read. Literal entries such as `{"value": "<literal>"}` or
// `{"custom": ...}` — and any non-list value — pass through untouched.

// asJSONArray reports whether raw is a JSON array and, if so, returns its elements.
func asJSONArray(raw json.RawMessage) ([]json.RawMessage, bool) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, false
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, false
	}
	return arr, true
}

// mapSectionArrays applies transform to every element of every list-valued section, leaving
// non-list sections untouched. It centralises the nested walk shared by the expand/collapse
// helpers so each only has to describe the per-element rewrite. transform returns the element
// to keep (unchanged when it doesn't apply).
func mapSectionArrays(raw json.RawMessage, transform func(json.RawMessage) (json.RawMessage, error)) (json.RawMessage, error) {
	var sections map[string]json.RawMessage
	if err := json.Unmarshal(raw, &sections); err != nil {
		return nil, err
	}
	for key, val := range sections {
		arr, ok := asJSONArray(val)
		if !ok {
			continue
		}
		for i, elem := range arr {
			out, err := transform(elem)
			if err != nil {
				return nil, err
			}
			arr[i] = out
		}
		merged, err := json.Marshal(arr)
		if err != nil {
			return nil, err
		}
		sections[key] = merged
	}
	return json.Marshal(sections)
}

// expandOrcaElement rewrites a bare JSON string into `{"orca": "<string>"}`. Any other shape
// (literal object, etc.) is returned unchanged.
func expandOrcaElement(elem json.RawMessage) (json.RawMessage, error) {
	var s string
	if json.Unmarshal(elem, &s) != nil {
		return elem, nil // not a bare string — leave literal objects untouched
	}
	return json.Marshal(map[string]string{"orca": s})
}

// expandOrcaShorthand rewrites bare JSON string list elements into `{"orca": "<string>"}`.
func expandOrcaShorthand(raw json.RawMessage) (json.RawMessage, error) {
	return mapSectionArrays(raw, expandOrcaElement)
}

// DecodeOrcaMappingField decodes a mapping_json state string and expands the bare-string
// shorthand into the API's `{"orca": ...}` wire form. Behaves like DecodeJSONField otherwise.
func DecodeOrcaMappingField(s jsonStringValue, fieldName string) (json.RawMessage, diag.Diagnostics) {
	raw, diags := DecodeJSONField(s, fieldName)
	if diags.HasError() || len(raw) == 0 {
		return raw, diags
	}
	expanded, err := expandOrcaShorthand(raw)
	if err != nil {
		diags.AddError(fmt.Sprintf("Invalid mapping in %s", fieldName), err.Error())
		return raw, diags
	}
	return expanded, diags
}

// EncodeOrcaMappingField stores the API's mapping value on state verbatim. No collapse is needed:
// OrcaMappingType's semantic equality (see orca_mapping_type.go) treats the API's `{"orca": ...}`
// wire form as equal to the user's bare-string shorthand, so the framework keeps the user's HCL
// form in state and no diff appears. An empty API value maps back to a null/planned value so an
// unset attribute does not diff against `{}`.
func EncodeOrcaMappingField(raw json.RawMessage, planned OrcaMapping) (OrcaMapping, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(raw) == 0 {
		if planned.IsNull() || planned.IsUnknown() {
			return NewOrcaMappingNull(), diags
		}
		return planned, diags
	}
	var generic interface{}
	if err := json.Unmarshal(raw, &generic); err != nil {
		diags.AddError("Invalid mapping from API", err.Error())
		return planned, diags
	}
	if isEmptyJSONValue(generic) {
		if planned.IsNull() || planned.IsUnknown() {
			return NewOrcaMappingNull(), diags
		}
		return planned, diags
	}
	return NewOrcaMappingValue(string(raw)), diags
}
