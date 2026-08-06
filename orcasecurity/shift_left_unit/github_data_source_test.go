package shift_left_unit

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGithubAccountsToListValue(t *testing.T) {
	accounts := []api_client.GithubInstallation{
		{ID: "i-1", AccountName: "acme", ScmUnitCommonFields: api_client.ScmUnitCommonFields{InstallationMode: "SCAN_ALL_INCLUDE_FUTURE"}},
		{ID: "i-2", AccountName: "beta"},
	}
	list, diags := githubAccountsSpec.ListValue(accounts)
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
	if obj.Attributes()["account_id"].(types.String).ValueString() != "i-1" {
		t.Errorf("account_id must mirror the Orca UUID: %v", obj.Attributes())
	}
}
