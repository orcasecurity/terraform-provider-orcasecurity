package integrations_common

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// jsonStringValue is any framework string value that carries JSON: types.String,
// jsontypes.Normalized, or OrcaMapping. Decoding only needs the null/unknown state and the raw
// string, so accepting the interface lets the same helper serve every mapping attribute type.
type jsonStringValue interface {
	IsNull() bool
	IsUnknown() bool
	ValueString() string
}

// DecodeJSONField turns a Terraform String holding JSON into json.RawMessage suitable for the
// API payload. Null / unknown / empty string all map to nil so json.Marshal with
// “omitempty“ skips the field. Any non-empty value is validated as JSON to surface bad
// input at plan time instead of an opaque 400 from Orca.
func DecodeJSONField(s jsonStringValue, fieldName string) (json.RawMessage, diag.Diagnostics) {
	var diags diag.Diagnostics
	if s.IsNull() || s.IsUnknown() || s.ValueString() == "" {
		return nil, diags
	}
	raw := json.RawMessage(s.ValueString())
	if !json.Valid(raw) {
		diags.AddError(fmt.Sprintf("Invalid JSON in %s", fieldName), "Value must be a JSON-encoded object.")
		return nil, diags
	}
	return raw, diags
}

// EncodeJSONField turns the json.RawMessage returned by the API into a jsontypes.Normalized for
// state. The value is stored verbatim — jsontypes.Normalized's semantic equality ignores
// whitespace and object key order, so plans don't drift on cosmetic differences, and the
// framework keeps the user's planned form in state when the two are semantically equal. Storing
// the raw bytes (rather than re-marshalling) also preserves number fidelity. Empty / nil API
// values preserve the user's planned shape (null stays null; an explicit empty value stays).
func EncodeJSONField(raw json.RawMessage, planned jsontypes.Normalized) (jsontypes.Normalized, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(raw) == 0 {
		if planned.IsNull() || planned.IsUnknown() {
			return jsontypes.NewNormalizedNull(), diags
		}
		return planned, diags
	}
	var generic interface{}
	if err := json.Unmarshal(raw, &generic); err != nil {
		diags.AddError("Invalid JSON from API", err.Error())
		return planned, diags
	}
	// The API may return an empty object/array/null to mean "no mapping set".
	// Treat that the same as an absent value so an unset (null) attribute does
	// not diff against a state of "{}". An explicit planned empty value is
	// preserved.
	if isEmptyJSONValue(generic) {
		if planned.IsNull() || planned.IsUnknown() {
			return jsontypes.NewNormalizedNull(), diags
		}
		return planned, diags
	}
	return jsontypes.NewNormalizedValue(string(raw)), diags
}

// isEmptyJSONValue reports whether a decoded JSON value carries no content:
// null, an empty object, or an empty array.
func isEmptyJSONValue(v interface{}) bool {
	switch t := v.(type) {
	case nil:
		return true
	case map[string]interface{}:
		return len(t) == 0
	case []interface{}:
		return len(t) == 0
	}
	return false
}

// JSONFieldDecode describes one (Terraform String -> API RawMessage) mapping: Src is the
// planned Terraform value, Field names the attribute for error messages, and Dst points at the
// API config field to fill in.
type JSONFieldDecode struct {
	Src   jsonStringValue
	Field string
	Dst   *json.RawMessage
}

// DecodeJSONFields runs DecodeJSONField over each mapping, appending any diagnostics and writing
// the resulting RawMessage into the field's Dst. It captures the loop shared by the per-variant
// template resources.
func DecodeJSONFields(fields []JSONFieldDecode, diags *diag.Diagnostics) {
	for _, f := range fields {
		raw, d := DecodeJSONField(f.Src, f.Field)
		diags.Append(d...)
		*f.Dst = raw
	}
}

// JSONFieldEncode describes one (API RawMessage -> Terraform String) mapping: Raw is the API
// value and Dst points at the Terraform state field, which also supplies the planned shape.
type JSONFieldEncode struct {
	Raw json.RawMessage
	Dst *jsontypes.Normalized
}

// EncodeJSONFields runs EncodeJSONField over each mapping, appending any diagnostics and writing
// the resulting String back into the field's Dst. It captures the loop shared by the per-variant
// template resources.
func EncodeJSONFields(fields []JSONFieldEncode, diags *diag.Diagnostics) {
	for _, f := range fields {
		encoded, d := EncodeJSONField(f.Raw, *f.Dst)
		diags.Append(d...)
		*f.Dst = encoded
	}
}
