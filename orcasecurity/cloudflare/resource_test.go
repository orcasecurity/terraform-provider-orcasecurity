package cloudflare

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func getSchema(t *testing.T) map[string]schema.Attribute {
	t.Helper()
	r := NewCloudflareResource()
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
func TestCloudflareMetadata_TypeNameSuffix(t *testing.T) {
	r := NewCloudflareResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "orcasecurity"}, resp)
	if resp.TypeName != "orcasecurity_integration_cloudflare" {
		t.Errorf("type name = %q, want orcasecurity_integration_cloudflare", resp.TypeName)
	}
}

// The schema must carry the common attributes plus the single api_token attribute, and omit
// business_units.
func TestCloudflareSchema_AttributesPresent(t *testing.T) {
	attrs := getSchema(t)
	for _, name := range []string{"id", "template_name", "is_enabled", "is_default", "api_token"} {
		if _, ok := attrs[name]; !ok {
			t.Errorf("expected attribute %q in schema", name)
		}
	}
	if _, ok := attrs["business_units"]; ok {
		t.Error("cloudflare does not support business_units; attribute must be absent")
	}
}

// api_token is the only credential and must be Required + Sensitive.
func TestCloudflareSchema_APITokenSensitive(t *testing.T) {
	attrs := getSchema(t)
	sa, ok := attrs["api_token"].(schema.StringAttribute)
	if !ok {
		t.Fatal("api_token not a string attribute")
	}
	if !sa.Required || !sa.Sensitive {
		t.Errorf("api_token must be Required and Sensitive, got %#v", sa)
	}
}

// Every schema attribute must have a matching tfsdk tag on the state model.
func TestCloudflareState_TagsCoverSchema(t *testing.T) {
	attrs := getSchema(t)
	tags := tfsdkTags(&state{})
	for name := range attrs {
		if _, ok := tags[name]; !ok {
			t.Errorf("state struct is missing a tfsdk tag for schema attribute %q", name)
		}
	}
}
