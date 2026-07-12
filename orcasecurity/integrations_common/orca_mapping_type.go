package integrations_common

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// OrcaMappingType is the attribute type for the `mapping_json` field. It behaves like
// jsontypes.Normalized — JSON compared by meaning, so whitespace and object key order never
// cause a diff — but its semantic-equality check additionally understands the bare-string orca
// shorthand: a list element written as "<field>" is treated as equal to the API's wire form
// {"orca":"<field>"}. So a plan stays stable regardless of which form the user writes in HCL
// (bare string, {"orca":...} object, heredoc, or unsorted keys), while state holds whatever the
// API returns.
type OrcaMappingType struct {
	jsontypes.NormalizedType
}

var _ basetypes.StringTypable = OrcaMappingType{}

func (t OrcaMappingType) String() string {
	return "integrations_common.OrcaMappingType"
}

func (t OrcaMappingType) ValueType(context.Context) attr.Value {
	return OrcaMapping{}
}

func (t OrcaMappingType) Equal(o attr.Type) bool {
	other, ok := o.(OrcaMappingType)
	if !ok {
		return false
	}
	return t.NormalizedType.Equal(other.NormalizedType)
}

func (t OrcaMappingType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return OrcaMapping{Normalized: jsontypes.Normalized{StringValue: in}}, nil
}

func (t OrcaMappingType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}
	stringValuable, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}
	return stringValuable, nil
}

// OrcaMapping is the value for OrcaMappingType. It embeds jsontypes.Normalized and overrides the
// identity and semantic-equality hooks so the shorthand is recognised.
type OrcaMapping struct {
	jsontypes.Normalized
}

var (
	_ basetypes.StringValuable                   = OrcaMapping{}
	_ basetypes.StringValuableWithSemanticEquals = OrcaMapping{}
	_ xattr.ValidateableAttribute                = OrcaMapping{}
)

func (v OrcaMapping) Type(context.Context) attr.Type {
	return OrcaMappingType{}
}

func (v OrcaMapping) Equal(o attr.Value) bool {
	other, ok := o.(OrcaMapping)
	if !ok {
		return false
	}
	return v.Normalized.Equal(other.Normalized)
}

// StringSemanticEquals expands the orca shorthand on both the state value (v) and the incoming
// config value, then delegates the whitespace/key-order-insensitive comparison to
// jsontypes.Normalized. Equal values keep the user's HCL form in state; only a real change
// produces a diff.
func (v OrcaMapping) StringSemanticEquals(ctx context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	newValue, ok := newValuable.(OrcaMapping)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			fmt.Sprintf("An unexpected value type was received while performing semantic equality checks.\n\nExpected: %T\nGot: %T", v, newValuable),
		)
		return false, diags
	}
	lhs := expandForCompare(v.ValueString())
	rhs := expandForCompare(newValue.ValueString())
	return jsontypes.NewNormalizedValue(lhs).StringSemanticEquals(ctx, jsontypes.NewNormalizedValue(rhs))
}

// expandForCompare rewrites the bare-string orca shorthand into its {"orca":...} wire form so
// two mappings can be compared on equal footing. Non-JSON or non-object input is returned
// unchanged, letting the downstream JSON comparison surface the error consistently.
func expandForCompare(s string) string {
	if s == "" {
		return s
	}
	raw := json.RawMessage(s)
	if !json.Valid(raw) {
		return s
	}
	expanded, err := expandOrcaShorthand(raw)
	if err != nil {
		return s
	}
	return string(expanded)
}

// NewOrcaMappingNull returns a null OrcaMapping.
func NewOrcaMappingNull() OrcaMapping {
	return OrcaMapping{Normalized: jsontypes.NewNormalizedNull()}
}

// NewOrcaMappingUnknown returns an unknown OrcaMapping.
func NewOrcaMappingUnknown() OrcaMapping {
	return OrcaMapping{Normalized: jsontypes.NewNormalizedUnknown()}
}

// NewOrcaMappingValue returns an OrcaMapping holding the given string.
func NewOrcaMappingValue(s string) OrcaMapping {
	return OrcaMapping{Normalized: jsontypes.NewNormalizedValue(s)}
}
