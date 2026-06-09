package shift_left_policy_catalog_controls

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestControlsToListValue(t *testing.T) {
	controls := []api_client.CatalogControlSummary{
		{ID: "ctrl-1", Title: "First", Category: "cat-a", Priority: "HIGH"},
		{ID: "ctrl-2", Title: "Second", Category: "cat-b", Priority: "LOW"},
	}

	list, diags := controlsToListValue(controls)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	elems := list.Elements()
	if len(elems) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(elems))
	}

	obj, ok := elems[0].(types.Object)
	if !ok {
		t.Fatalf("expected object element, got %T", elems[0])
	}
	attrs := obj.Attributes()
	if got := attrs["id"].(types.String).ValueString(); got != "ctrl-1" {
		t.Errorf("expected id ctrl-1, got %s", got)
	}
	if got := attrs["title"].(types.String).ValueString(); got != "First" {
		t.Errorf("expected title First, got %s", got)
	}
	if got := attrs["category"].(types.String).ValueString(); got != "cat-a" {
		t.Errorf("expected category cat-a, got %s", got)
	}
	if got := attrs["priority"].(types.String).ValueString(); got != "HIGH" {
		t.Errorf("expected priority HIGH, got %s", got)
	}
}

func TestControlsToListValue_Empty(t *testing.T) {
	list, diags := controlsToListValue(nil)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(list.Elements()) != 0 {
		t.Errorf("expected empty list, got %d elements", len(list.Elements()))
	}
}
