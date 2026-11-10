package shift_left_projects

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestProjectsToListValue(t *testing.T) {
	projects := []api_client.ShiftLeftProjectSummary{
		{ID: "p-1", Name: "allscan", Key: "allscan"},
		{ID: "p-2", Name: "backend", Key: "backend"},
	}
	list, diags := projectsToListValue(projects)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if len(list.Elements()) != 2 {
		t.Fatalf("expected 2, got %d", len(list.Elements()))
	}
	obj := list.Elements()[0].(types.Object)
	if obj.Attributes()["id"].(types.String).ValueString() != "p-1" {
		t.Errorf("bad id: %v", obj.Attributes())
	}
	if obj.Attributes()["name"].(types.String).ValueString() != "allscan" {
		t.Errorf("bad name: %v", obj.Attributes())
	}
	if obj.Attributes()["key"].(types.String).ValueString() != "allscan" {
		t.Errorf("bad key: %v", obj.Attributes())
	}
}

func TestProjectsToListValue_Empty(t *testing.T) {
	list, diags := projectsToListValue(nil)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if len(list.Elements()) != 0 {
		t.Errorf("expected 0 elements, got %d", len(list.Elements()))
	}
}
