package shift_left_azure_devops_account

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAccountsToListValue(t *testing.T) {
	accs := []api_client.AzureDevopsAccount{
		{ID: "a-1", InstallationID: "i-1", AccountName: "acme", InstallationMode: "SCAN_ALL_INCLUDE_FUTURE"},
		{ID: "a-2", InstallationID: "i-1", AccountName: "beta"},
	}
	list, diags := accountsToListValue(accs)
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
	if obj.Attributes()["account_id"].(types.String).ValueString() != "a-1" {
		t.Errorf("bad account_id: %v", obj.Attributes())
	}
	if obj.Attributes()["installation_id"].(types.String).ValueString() != "i-1" {
		t.Errorf("bad installation_id: %v", obj.Attributes())
	}
}
