package orcasecurity

// Terraform core only started carrying prior state forward for unchanged Computed attributes in
// 1.4. On 1.0 through 1.3 — which this provider still supports and CI still exercises — a Computed
// attribute without a carry-forward plan modifier replans as "known after apply" on every run, so
// `terraform apply` never reaches an empty plan and the resource looks permanently dirty.
//
// That is easy to reintroduce one attribute at a time and impossible to catch on a modern
// Terraform, so this test walks the shift-left SCM resource schemas directly instead of relying on
// the version matrix to notice.

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// The SCM connection/unit/repository resources (plus the org-wide SCM posture
// default-policy singleton), whose schemas drift together. Listed explicitly so
// that renaming a resource out of the set is a deliberate edit rather than a
// silent loss of coverage.
var shiftLeftScmResourceTypes = []string{
	"orcasecurity_shift_left_azure_devops_account",
	"orcasecurity_shift_left_azure_devops_installation",
	"orcasecurity_shift_left_azure_devops_repository",
	"orcasecurity_shift_left_bitbucket_account",
	"orcasecurity_shift_left_bitbucket_installation",
	"orcasecurity_shift_left_bitbucket_repository",
	"orcasecurity_shift_left_github_account",
	"orcasecurity_shift_left_github_repository",
	"orcasecurity_shift_left_gitlab_group",
	"orcasecurity_shift_left_gitlab_installation",
	"orcasecurity_shift_left_gitlab_repository",
	"orcasecurity_shift_left_scm_posture_default_policy",
}

func TestShiftLeftScmSchemasSettle(t *testing.T) {
	ctx := context.Background()
	want := make(map[string]bool, len(shiftLeftScmResourceTypes))
	for _, typeName := range shiftLeftScmResourceTypes {
		want[typeName] = true
	}

	seen := map[string]bool{}
	for _, newResource := range New("test")().Resources(ctx) {
		res := newResource()

		var meta resource.MetadataResponse
		res.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "orcasecurity"}, &meta)
		if !want[meta.TypeName] {
			continue
		}
		seen[meta.TypeName] = true

		var schemaResp resource.SchemaResponse
		res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
		if schemaResp.Diagnostics.HasError() {
			t.Errorf("%s: schema returned diagnostics: %v", meta.TypeName, schemaResp.Diagnostics)
			continue
		}
		assertComputedAttributesCarryForward(t, meta.TypeName, "", schemaResp.Schema.Attributes)
	}

	var missing []string
	for typeName := range want {
		if !seen[typeName] {
			missing = append(missing, typeName)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("these resources are no longer registered under the expected type names, so they went unchecked: %s",
			strings.Join(missing, ", "))
	}
}

func assertComputedAttributesCarryForward(t *testing.T, typeName, prefix string, attrs map[string]rschema.Attribute) {
	t.Helper()
	for name, attr := range attrs {
		attrPath := prefix + name
		switch nested := attr.(type) {
		case rschema.SingleNestedAttribute:
			assertComputedAttributesCarryForward(t, typeName, attrPath+".", nested.Attributes)
		case rschema.ListNestedAttribute:
			assertComputedAttributesCarryForward(t, typeName, attrPath+".", nested.NestedObject.Attributes)
		case rschema.SetNestedAttribute:
			assertComputedAttributesCarryForward(t, typeName, attrPath+".", nested.NestedObject.Attributes)
		case rschema.MapNestedAttribute:
			assertComputedAttributesCarryForward(t, typeName, attrPath+".", nested.NestedObject.Attributes)
		}
		if !attr.IsComputed() {
			continue
		}
		modifiers, inspectable := planModifiers(attr)
		if !inspectable {
			t.Errorf("%s: %s is a %T, which this test cannot inspect — add it to planModifiers",
				typeName, attrPath, attr)
			continue
		}
		// RequiresReplace alone is not enough: TF 1.0–1.3 still replans the
		// attribute as "known after apply" forever without a carry-forward
		// modifier (UseStateForUnknown / ProjectIDPlanModifier / …) or an
		// explicit volatile marker (ComputedVolatile*).
		if !hasAcceptableComputedModifier(modifiers) {
			t.Errorf("%s: computed attribute %q has no carry-forward or volatile plan modifier, "+
				"so Terraform 1.0-1.3 replans it as \"known after apply\" on every apply and the plan never settles; "+
				"declare it with shift_left_common.Computed*/OptionalComputed* or shift_left_integration.ComputedVolatile*", typeName, attrPath)
		}
	}
}

// hasAcceptableComputedModifier is true when at least one modifier is not a
// RequiresReplace* variant (carry-forward OR explicit volatile). len(PlanModifiers) > 0
// used to pass a RequiresReplace-only attribute that never settles on TF 1.0–1.3.
func hasAcceptableComputedModifier(modifiers []any) bool {
	for _, m := range modifiers {
		name := fmt.Sprintf("%T", m)
		if strings.Contains(name, "requiresReplace") {
			continue
		}
		return true
	}
	return false
}

func planModifiers(attr rschema.Attribute) (modifiers []any, inspectable bool) {
	switch a := attr.(type) {
	case rschema.StringAttribute:
		return asAny(a.PlanModifiers), true
	case rschema.BoolAttribute:
		return asAny(a.PlanModifiers), true
	case rschema.Int64Attribute:
		return asAny(a.PlanModifiers), true
	case rschema.Float64Attribute:
		return asAny(a.PlanModifiers), true
	case rschema.NumberAttribute:
		return asAny(a.PlanModifiers), true
	case rschema.SetAttribute:
		return asAny(a.PlanModifiers), true
	case rschema.ListAttribute:
		return asAny(a.PlanModifiers), true
	case rschema.MapAttribute:
		return asAny(a.PlanModifiers), true
	case rschema.ObjectAttribute:
		return asAny(a.PlanModifiers), true
	case rschema.SingleNestedAttribute:
		return asAny(a.PlanModifiers), true
	case rschema.ListNestedAttribute:
		return asAny(a.PlanModifiers), true
	case rschema.SetNestedAttribute:
		return asAny(a.PlanModifiers), true
	case rschema.MapNestedAttribute:
		return asAny(a.PlanModifiers), true
	default:
		return nil, false
	}
}

func asAny[T any](in []T) []any {
	out := make([]any, len(in))
	for i := range in {
		out[i] = in[i]
	}
	return out
}
