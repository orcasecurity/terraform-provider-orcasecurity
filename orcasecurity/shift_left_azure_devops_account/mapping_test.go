package shift_left_azure_devops_account

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestApiToState_MapsAccountName(t *testing.T) {
	inst := &api_client.AzureDevopsAccount{
		ID: "abc", InstallationID: "inst-1", AccountName: "acme",
		ScmUnitCommonFields: api_client.ScmUnitCommonFields{
			InstallationMode:  "SCAN_ALL_INCLUDE_FUTURE",
			IntegrationStatus: "DISABLED_DUE_TO_INVALID_TOKEN",
			Policies:          []api_client.ScmPolicyRef{{ID: "pol-1"}},
		},
	}
	st := apiToState(inst)
	if st.ID.ValueString() != "abc" || st.AccountName.ValueString() != "acme" {
		t.Errorf("id/account_name mismatch: %+v", st)
	}
	if st.InstallationID.ValueString() != "inst-1" {
		t.Errorf("installation_id mismatch: %+v", st)
	}
	elems := st.PoliciesIds.Elements()
	if len(elems) != 1 || elems[0].(types.String).ValueString() != "pol-1" {
		t.Errorf("policies_ids wrong: %+v", st.PoliciesIds)
	}
	if st.IntegrationStatus.ValueString() != "DISABLED_DUE_TO_INVALID_TOKEN" {
		t.Errorf("integration_status: got %v", st.IntegrationStatus)
	}
	if !st.ProjectID.IsNull() {
		t.Errorf("unbound project_id must be null, got %#v", st.ProjectID)
	}
}
