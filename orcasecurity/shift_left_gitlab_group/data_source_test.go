package shift_left_gitlab_group

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGroupsToListValue(t *testing.T) {
	grps := []api_client.GitlabGroup{
		{ID: "g-1", InstallationID: "i-1", AccountName: "acme", GitlabGroupID: 11, ScmUnitCommonFields: api_client.ScmUnitCommonFields{InstallationMode: "SCAN_ALL_INCLUDE_FUTURE"}},
		{ID: "g-2", InstallationID: "i-1", AccountName: "beta", GitlabGroupID: 22},
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
	if obj.Attributes()["id"].(types.String).ValueString() != "g-1" {
		t.Errorf("bad id: %v", obj.Attributes())
	}
	if obj.Attributes()["gitlab_group_id"].(types.Int64).ValueInt64() != 11 {
		t.Errorf("bad gitlab_group_id: %v", obj.Attributes())
	}
	if obj.Attributes()["installation_id"].(types.String).ValueString() != "i-1" {
		t.Errorf("bad installation_id: %v", obj.Attributes())
	}
}
