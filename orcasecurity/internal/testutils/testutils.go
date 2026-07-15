// Package testutils holds test-only helpers shared by the per-package unit tests. It is imported
// exclusively from _test.go files, so nothing here ships in the provider binary.
package testutils

import (
	"context"
	"reflect"
	"testing"
	"unsafe"

	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ResourceSchemaAttrs drives the resource's Schema method the way the framework does and returns
// the resulting attribute map. No API client is needed to build a schema.
func ResourceSchemaAttrs(t *testing.T, r resource.Resource) map[string]schema.Attribute {
	t.Helper()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema build diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema.Attributes
}

// TfsdkTags walks a struct (including embedded structs) and collects every tfsdk tag. This lets a
// test assert the state model can actually decode the attributes the schema declares.
func TfsdkTags(v interface{}) map[string]struct{} {
	out := map[string]struct{}{}
	var walk func(rt reflect.Type)
	walk = func(rt reflect.Type) {
		if rt.Kind() == reflect.Ptr {
			rt = rt.Elem()
		}
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if f.Anonymous {
				walk(f.Type)
				continue
			}
			if tag, ok := f.Tag.Lookup("tfsdk"); ok {
				out[tag] = struct{}{}
			}
		}
	}
	walk(reflect.TypeOf(v))
	return out
}

// VariantResourceSpec describes the schema contract of a config-integration variant resource so
// CheckVariantResource can verify it uniformly across packages.
type VariantResourceSpec struct {
	NewResource func() resource.Resource
	// TypeName is the fully qualified resource type expected from Metadata.
	TypeName string
	// Secrets are string attributes that must be Required and Sensitive.
	Secrets []string
	// PlainRequired are string attributes that must be Required and non-sensitive.
	PlainRequired []string
	// Forbidden attributes must be absent (e.g. business_units on no-BU variants).
	Forbidden []string
	// State is a pointer to the variant's state struct; every schema attribute must have a
	// matching tfsdk tag on it, otherwise the framework can't decode plan/state and
	// BuildPayload/Extract would read zero values.
	State interface{}
}

// CheckVariantResource asserts the variant's Metadata-derived type name, the presence of the
// common attributes plus every declared credential, credential sensitivity, forbidden attributes,
// and state-tag coverage of the schema.
func CheckVariantResource(t *testing.T, spec VariantResourceSpec) {
	t.Helper()
	r := spec.NewResource()

	mdResp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "orcasecurity"}, mdResp)
	if mdResp.TypeName != spec.TypeName {
		t.Errorf("type name = %q, want %q", mdResp.TypeName, spec.TypeName)
	}

	attrs := ResourceSchemaAttrs(t, r)
	present := []string{"id", "template_name", "is_enabled", "is_default"}
	present = append(present, spec.Secrets...)
	present = append(present, spec.PlainRequired...)
	checkAttrsPresent(t, attrs, present)
	checkAttrsAbsent(t, attrs, spec.Forbidden)
	checkRequiredStrings(t, attrs, spec.Secrets, true)
	checkRequiredStrings(t, attrs, spec.PlainRequired, false)
	checkStateTagCoverage(t, attrs, spec.State)
}

func checkAttrsPresent(t *testing.T, attrs map[string]schema.Attribute, names []string) {
	t.Helper()
	for _, name := range names {
		if _, ok := attrs[name]; !ok {
			t.Errorf("expected attribute %q in schema", name)
		}
	}
}

func checkAttrsAbsent(t *testing.T, attrs map[string]schema.Attribute, names []string) {
	t.Helper()
	for _, name := range names {
		if _, ok := attrs[name]; ok {
			t.Errorf("attribute %q must be absent from this variant's schema", name)
		}
	}
}

// checkRequiredStrings asserts each named attribute is a Required string whose Sensitive flag
// matches wantSensitive.
func checkRequiredStrings(t *testing.T, attrs map[string]schema.Attribute, names []string, wantSensitive bool) {
	t.Helper()
	kind := "non-sensitive"
	if wantSensitive {
		kind = "Sensitive"
	}
	for _, name := range names {
		sa, ok := attrs[name].(schema.StringAttribute)
		if !ok {
			t.Errorf("%s not a string attribute", name)
			continue
		}
		if !sa.Required || sa.Sensitive != wantSensitive {
			t.Errorf("%s must be Required and %s, got %#v", name, kind, sa)
		}
	}
}

func checkStateTagCoverage(t *testing.T, attrs map[string]schema.Attribute, state interface{}) {
	t.Helper()
	tags := TfsdkTags(state)
	for name := range attrs {
		if _, ok := tags[name]; !ok {
			t.Errorf("state struct is missing a tfsdk tag for schema attribute %q", name)
		}
	}
}

// StringList builds a types.List of strings, failing the test on a build error.
func StringList(t *testing.T, elems ...string) types.List {
	t.Helper()
	vals := make([]attr.Value, 0, len(elems))
	for _, e := range elems {
		vals = append(vals, types.StringValue(e))
	}
	lv, d := types.ListValue(types.StringType, vals)
	if d.HasError() {
		t.Fatalf("list build: %v", d)
	}
	return lv
}

// StringSet builds a types.Set of strings (e.g. the `business_units` attribute shape).
func StringSet(t *testing.T, elems ...string) types.Set {
	t.Helper()
	vals := make([]attr.Value, 0, len(elems))
	for _, e := range elems {
		vals = append(vals, types.StringValue(e))
	}
	s, d := types.SetValue(types.StringType, vals)
	if d.HasError() {
		t.Fatalf("set build: %v", d)
	}
	return s
}

// SameElements reports whether two string slices contain the same elements, ignoring order.
func SameElements(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := map[string]int{}
	for _, x := range a {
		counts[x]++
	}
	for _, x := range b {
		counts[x]--
	}
	for _, c := range counts {
		if c != 0 {
			return false
		}
	}
	return true
}

// SpecFromResource pulls the (unexported) Spec back out of a config-integration variant resource
// so in-package unit tests can exercise the variant's BuildPayload / Extract closures directly.
// The closures are anonymous and captured privately inside the generic skeleton; reflection +
// unsafe is the only way to reach them without adding a production-only accessor.
func SpecFromResource[P any](r resource.Resource) cc.Spec[P] {
	field := reflect.ValueOf(r).Elem().FieldByName("spec")
	field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
	return field.Interface().(cc.Spec[P])
}
