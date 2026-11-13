package integrations_common

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringList(values ...string) types.List {
	elems := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, types.StringValue(v))
	}
	return types.ListValueMust(types.StringType, elems)
}

func TestValidateShiftleftGlobalScope_AloneIsValid(t *testing.T) {
	diags := ValidateShiftleftGlobalScope(types.BoolValue(true), types.BoolValue(false), map[string]types.List{
		"cloud_accounts":     types.ListNull(types.StringType),
		"shiftleft_projects": stringList(),
		"user_filters":       types.ListUnknown(types.StringType),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
}

func TestValidateShiftleftGlobalScope_FalseNeverConflicts(t *testing.T) {
	diags := ValidateShiftleftGlobalScope(types.BoolValue(false), types.BoolValue(true), map[string]types.List{
		"cloud_accounts": stringList("ca-1"),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
}

func TestValidateShiftleftGlobalScope_UnknownIsSkipped(t *testing.T) {
	diags := ValidateShiftleftGlobalScope(types.BoolUnknown(), types.BoolValue(true), nil)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
}

func TestValidateShiftleftGlobalScope_ConflictsAreListed(t *testing.T) {
	diags := ValidateShiftleftGlobalScope(types.BoolValue(true), types.BoolValue(true), map[string]types.List{
		"cloud_accounts":     stringList(),
		"shiftleft_projects": stringList("sl-1"),
		"user_filters":       types.ListNull(types.StringType),
	})
	if !diags.HasError() {
		t.Fatal("expected a conflict")
	}
	detail := diags[0].Detail()
	if !strings.Contains(detail, "all_cloud_accounts, shiftleft_projects") {
		t.Errorf("conflict detail should name the conflicting scopes in order, got %q", detail)
	}
	if strings.Contains(detail, "with cloud_accounts") || strings.Contains(detail, "user_filters") {
		t.Errorf("empty and null lists must not be reported, got %q", detail)
	}
}
