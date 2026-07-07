package integrations_common

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The Orca field-mapping config for template integrations (jira, sn_incidents, monday, …) is a
// JSON object whose values are lists of entries. The common entry is a "pull this Orca alert
// field" reference, which the API represents verbatim as `{"orca": "<field>"}`. Writing that
// wrapper for every field is noisy in HCL, so these helpers let users write a bare string
// (`"<field>"`) instead: the provider expands it to `{"orca": "<field>"}` on the way to the API
// and collapses it back to a bare string on read, so plans don't drift. Literal entries such as
// `{"value": "<literal>"}` or `{"custom": ...}` — and any non-list value — pass through untouched.

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

// expandOrcaShorthand rewrites bare JSON string list elements into `{"orca": "<string>"}`.
func expandOrcaShorthand(raw json.RawMessage) (json.RawMessage, error) {
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
			var s string
			if json.Unmarshal(elem, &s) != nil {
				continue // not a bare string — leave literal objects untouched
			}
			wrapped, err := json.Marshal(map[string]string{"orca": s})
			if err != nil {
				return nil, err
			}
			arr[i] = wrapped
		}
		merged, err := json.Marshal(arr)
		if err != nil {
			return nil, err
		}
		sections[key] = merged
	}
	return json.Marshal(sections)
}

// collapseOrcaShorthand rewrites `{"orca": "<string>"}` list elements back into bare strings.
func collapseOrcaShorthand(raw json.RawMessage) (json.RawMessage, error) {
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
			var obj map[string]json.RawMessage
			if json.Unmarshal(elem, &obj) != nil {
				continue // not an object (already a bare string, etc.)
			}
			orcaRaw, ok := obj["orca"]
			if !ok || len(obj) != 1 {
				continue // literal ({"value": ...}) or multi-key — keep as object
			}
			var s string
			if json.Unmarshal(orcaRaw, &s) != nil {
				continue // orca value is not a plain string — keep as object
			}
			bare, err := json.Marshal(s)
			if err != nil {
				return nil, err
			}
			arr[i] = bare
		}
		merged, err := json.Marshal(arr)
		if err != nil {
			return nil, err
		}
		sections[key] = merged
	}
	return json.Marshal(sections)
}

// DecodeOrcaMappingField decodes a mapping_json state string and expands the bare-string
// shorthand into the API's `{"orca": ...}` wire form. Behaves like DecodeJSONField otherwise.
func DecodeOrcaMappingField(s types.String, fieldName string) (json.RawMessage, diag.Diagnostics) {
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

// EncodeOrcaMappingField collapses the API's `{"orca": ...}` wire form back into the bare-string
// shorthand, then normalizes to compact JSON for state. Behaves like EncodeJSONField otherwise.
func EncodeOrcaMappingField(raw json.RawMessage, planned types.String) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(raw) == 0 {
		return EncodeJSONField(raw, planned)
	}
	collapsed, err := collapseOrcaShorthand(raw)
	if err != nil {
		diags.AddError("Invalid mapping from API", err.Error())
		return planned, diags
	}
	return EncodeJSONField(collapsed, planned)
}
