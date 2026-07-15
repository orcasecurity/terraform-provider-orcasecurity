package akamai

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// getSchema drives the resource's Schema method the way the framework does and returns the
// resulting attribute map. No API client is needed to build a schema.
func getSchema(t *testing.T) map[string]schema.Attribute {
	t.Helper()
	r := NewAkamaiResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema build diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema.Attributes
}

// tfsdkTags walks a struct (including embedded structs) and collects every tfsdk tag. This
// lets a test assert the state model can actually decode the attributes the schema declares.
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
func TestAkamaiMetadata_TypeNameSuffix(t *testing.T) {
	r := NewAkamaiResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "orcasecurity"}, resp)
	if resp.TypeName != "orcasecurity_integration_akamai" {
		t.Errorf("type name = %q, want orcasecurity_integration_akamai", resp.TypeName)
	}
}

// The schema must carry the common attributes plus every Akamai-specific attribute, and must
// omit business_units (akamai uses the no-BU CommonFields flavour).
func TestAkamaiSchema_AttributesPresent(t *testing.T) {
	attrs := getSchema(t)
	for _, name := range []string{
		"id", "template_name", "is_enabled", "is_default",
		"access_token", "client_token", "client_secret", "host",
	} {
		if _, ok := attrs[name]; !ok {
			t.Errorf("expected attribute %q in schema", name)
		}
	}
	if _, ok := attrs["business_units"]; ok {
		t.Error("akamai does not support business_units; attribute must be absent")
	}
}

// The three EdgeGrid credentials are secrets and required; host is required but not sensitive.
func TestAkamaiSchema_CredentialSensitivity(t *testing.T) {
	attrs := getSchema(t)
	for _, name := range []string{"access_token", "client_token", "client_secret"} {
		sa, ok := attrs[name].(schema.StringAttribute)
		if !ok {
			t.Fatalf("%s not a string attribute", name)
		}
		if !sa.Sensitive {
			t.Errorf("%s must be marked Sensitive", name)
		}
		if !sa.Required {
			t.Errorf("%s must be Required", name)
		}
	}
	host, ok := attrs["host"].(schema.StringAttribute)
	if !ok || !host.Required || host.Sensitive {
		t.Errorf("host must be Required and non-sensitive, got %#v", attrs["host"])
	}
}

// Every schema attribute must have a matching tfsdk tag on the state model, otherwise the
// framework can't decode plan/state and BuildPayload/Extract would read zero values.
func TestAkamaiState_TagsCoverSchema(t *testing.T) {
	attrs := getSchema(t)
	tags := tfsdkTags(&state{})
	for name := range attrs {
		if _, ok := tags[name]; !ok {
			t.Errorf("state struct is missing a tfsdk tag for schema attribute %q", name)
		}
	}
}
