package zscaler

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func getSchema(t *testing.T) map[string]schema.Attribute {
	t.Helper()
	r := NewZscalerResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema build diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema.Attributes
}

func tfsdkTags(v interface{}) map[string]struct{} {
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

// Metadata must derive the type name by appending the variant suffix to the provider prefix.
func TestZscalerMetadata_TypeNameSuffix(t *testing.T) {
	r := NewZscalerResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "orcasecurity"}, resp)
	if resp.TypeName != "orcasecurity_integration_zscaler_zpa" {
		t.Errorf("type name = %q, want orcasecurity_integration_zscaler_zpa", resp.TypeName)
	}
}

// The schema must carry the common attributes plus the Zscaler-specific attributes, and omit
// business_units.
func TestZscalerSchema_AttributesPresent(t *testing.T) {
	attrs := getSchema(t)
	for _, name := range []string{
		"id", "template_name", "is_enabled", "is_default",
		"vanity_domain", "client_id", "client_secret",
	} {
		if _, ok := attrs[name]; !ok {
			t.Errorf("expected attribute %q in schema", name)
		}
	}
	if _, ok := attrs["business_units"]; ok {
		t.Error("zscaler does not support business_units; attribute must be absent")
	}
}

// The OAuth client_id/client_secret are secrets; vanity_domain is a plain identifier.
func TestZscalerSchema_CredentialSensitivity(t *testing.T) {
	attrs := getSchema(t)
	for _, name := range []string{"client_id", "client_secret"} {
		sa, ok := attrs[name].(schema.StringAttribute)
		if !ok {
			t.Fatalf("%s not a string attribute", name)
		}
		if !sa.Sensitive || !sa.Required {
			t.Errorf("%s must be Required and Sensitive, got %#v", name, sa)
		}
	}
	vd, ok := attrs["vanity_domain"].(schema.StringAttribute)
	if !ok || !vd.Required || vd.Sensitive {
		t.Errorf("vanity_domain must be Required and non-sensitive, got %#v", attrs["vanity_domain"])
	}
}

// Every schema attribute must have a matching tfsdk tag on the state model.
func TestZscalerState_TagsCoverSchema(t *testing.T) {
	attrs := getSchema(t)
	tags := tfsdkTags(&state{})
	for name := range attrs {
		if _, ok := tags[name]; !ok {
			t.Errorf("state struct is missing a tfsdk tag for schema attribute %q", name)
		}
	}
}
