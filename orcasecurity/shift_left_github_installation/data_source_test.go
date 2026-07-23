package shift_left_github_installation

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestInstallationsToListValue(t *testing.T) {
	insts := []api_client.GithubInstallation{
		{ID: "i-1", AccountName: "acme", InstallationMode: "SCAN_ALL_INCLUDE_FUTURE"},
		{ID: "i-2", AccountName: "beta"},
	}
	list, diags := installationsToListValue(insts)
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
}
