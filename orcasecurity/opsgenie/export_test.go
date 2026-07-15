package opsgenie

import (
	"context"
	"reflect"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"unsafe"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// specFromResource pulls the (unexported) Spec back out of the resource returned by
// NewOpsgenieResource so the in-package unit tests can exercise the variant's BuildPayload /
// Extract closures directly. The closures are anonymous and captured privately inside the
// generic skeleton; reflection + unsafe is the only way to reach them without adding a
// production-only accessor. This lives in a _test.go file so it never ships in the provider.
func specFromResource() cc.Spec[api_client.OpsgenieExternalServiceConfig] {
	r := NewOpsgenieResource()
	field := reflect.ValueOf(r).Elem().FieldByName("spec")
	field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
	return field.Interface().(cc.Spec[api_client.OpsgenieExternalServiceConfig])
}

// buildPayload / extract are thin wrappers so tests read like the slack/monday mapping tests.
func buildPayload(ctx context.Context, st cc.State, diags *diag.Diagnostics) api_client.OpsgenieExternalServiceConfig {
	return specFromResource().BuildPayload(ctx, st, diags)
}

func extract(o *api_client.OpsgenieExternalServiceConfig, st cc.State, diags *diag.Diagnostics) cc.APIObject {
	return specFromResource().Extract(o, st, diags)
}
