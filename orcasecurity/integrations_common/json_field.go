package integrations_common

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DecodeJSONField turns a Terraform String holding JSON into json.RawMessage suitable for the
// API payload. Null / unknown / empty string all map to nil so json.Marshal with
// “omitempty“ skips the field. Any non-empty value is validated as JSON to surface bad
// input at plan time instead of an opaque 400 from Orca.
func DecodeJSONField(s types.String, fieldName string) (json.RawMessage, diag.Diagnostics) {
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

// EncodeJSONField turns the json.RawMessage returned by the API into a Terraform String for
// state. The output is re-marshalled to compact JSON so plans don't drift on whitespace
// differences between the API response and the user's HCL. Empty / nil API values preserve
// the user's planned shape (null stays null; an explicit empty string stays empty).
func EncodeJSONField(raw json.RawMessage, planned types.String) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(raw) == 0 {
		if planned.IsNull() || planned.IsUnknown() {
			return types.StringNull(), diags
		}
		return planned, diags
	}
	var generic interface{}
	if err := json.Unmarshal(raw, &generic); err != nil {
		diags.AddError("Invalid JSON from API", err.Error())
		return planned, diags
	}
	encoded, err := json.Marshal(generic)
	if err != nil {
		diags.AddError("Could not re-marshal JSON from API", err.Error())
		return planned, diags
	}
	return types.StringValue(string(encoded)), diags
}
