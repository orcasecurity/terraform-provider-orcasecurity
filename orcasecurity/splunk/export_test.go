package splunk

import (
	"context"
	"reflect"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"unsafe"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// specFromResource pulls the (unexported) Spec back out of NewSplunkResource so the in-package
// unit tests can exercise the variant's BuildPayload / Extract closures directly. Reflection +
// unsafe is the only way to reach the anonymous closures without adding a production-only
// accessor; this file is a _test.go so it never ships in the provider.
func specFromResource() cc.Spec[api_client.SplunkExternalServiceConfig] {
	r := NewSplunkResource()
	field := reflect.ValueOf(r).Elem().FieldByName("spec")
	field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
	return field.Interface().(cc.Spec[api_client.SplunkExternalServiceConfig])
}

func buildPayload(ctx context.Context, st cc.State, diags *diag.Diagnostics) api_client.SplunkExternalServiceConfig {
	return specFromResource().BuildPayload(ctx, st, diags)
}

func extract(o *api_client.SplunkExternalServiceConfig, st cc.State, diags *diag.Diagnostics) cc.APIObject {
	return specFromResource().Extract(o, st, diags)
}
