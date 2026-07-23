package shift_left_gitlab_group

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGroupsToListValue(t *testing.T) {
	grps := []api_client.GitlabGroup{
		{ID: "g-1", InstallationID: "i-1", AccountName: "acme", InstallationMode: "SCAN_ALL_INCLUDE_FUTURE"},
		{ID: "g-2", InstallationID: "i-1", AccountName: "beta"},
	}
	list, diags := groupsToListValue(grps)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if len(list.Elements()) != 2 {
		t.Fatalf("expected 2, got %d", len(list.Elements()))
	}
	obj := list.Elements()[0].(types.Object)
	if obj.Attributes()["account_name"].(types.String).ValueString() != "acme" {
		t.Errorf("bad account_name: %v", obj.Attributes())
	}
	if obj.Attributes()["group_id"].(types.String).ValueString() != "g-1" {
		t.Errorf("bad group_id: %v", obj.Attributes())
	}
	if obj.Attributes()["installation_id"].(types.String).ValueString() != "i-1" {
		t.Errorf("bad installation_id: %v", obj.Attributes())
	}
}
