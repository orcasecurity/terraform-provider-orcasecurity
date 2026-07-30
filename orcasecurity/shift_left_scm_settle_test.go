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
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// The SCM connection/unit/repository resources, whose schemas are generated from shared builders
// and therefore drift together. Listed explicitly so that renaming a resource out of the set is a
// deliberate edit rather than a silent loss of coverage.
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
		if nested, ok := attr.(rschema.SingleNestedAttribute); ok {
			assertComputedAttributesCarryForward(t, typeName, attrPath+".", nested.Attributes)
		}
		if !attr.IsComputed() {
			continue
		}
		modifiers, inspectable := planModifierCount(attr)
		if !inspectable {
			t.Errorf("%s: %s is a %T, which this test cannot inspect — add it to planModifierCount",
				typeName, attrPath, attr)
			continue
		}
		if modifiers == 0 {
			t.Errorf("%s: computed attribute %q has no plan modifier, so Terraform 1.0-1.3 replans it as "+
				"\"known after apply\" on every apply and the plan never settles; declare it with the "+
				"shift_left_integration.Computed*/OptionalComputed* helpers", typeName, attrPath)
		}
	}
}

func planModifierCount(attr rschema.Attribute) (count int, inspectable bool) {
	switch a := attr.(type) {
	case rschema.StringAttribute:
		return len(a.PlanModifiers), true
	case rschema.BoolAttribute:
		return len(a.PlanModifiers), true
	case rschema.Int64Attribute:
		return len(a.PlanModifiers), true
	case rschema.Float64Attribute:
		return len(a.PlanModifiers), true
	case rschema.NumberAttribute:
		return len(a.PlanModifiers), true
	case rschema.SetAttribute:
		return len(a.PlanModifiers), true
	case rschema.ListAttribute:
		return len(a.PlanModifiers), true
	case rschema.MapAttribute:
		return len(a.PlanModifiers), true
	case rschema.ObjectAttribute:
		return len(a.PlanModifiers), true
	case rschema.SingleNestedAttribute:
		return len(a.PlanModifiers), true
	default:
		return 0, false
	}
}
