package shift_left_github_installation

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestApiToState_MirrorsInstallationID(t *testing.T) {
	inst := &api_client.GithubInstallation{
		ID: "abc", AccountName: "acme",
		ScmUnitCommonFields: api_client.ScmUnitCommonFields{
			InstallationMode:  "SCAN_ALL_INCLUDE_FUTURE",
			IntegrationStatus: "ENABLED",
			Policies:          []api_client.ScmPolicyRef{{ID: "pol-1"}},
		},
	}
	st := apiToState(inst)
	if st.ID.ValueString() != "abc" || st.InstallationID.ValueString() != "abc" {
		t.Errorf("id/installation_id mismatch: %+v", st)
	}
	elems := st.PoliciesIds.Elements()
	if len(elems) != 1 || elems[0].(types.String).ValueString() != "pol-1" {
		t.Errorf("policies_ids wrong: %+v", st.PoliciesIds)
	}
	if st.IntegrationStatus.ValueString() != "ENABLED" {
		t.Errorf("integration_status: got %v", st.IntegrationStatus)
	}
	if !st.ProjectID.IsNull() {
		t.Errorf("unbound project_id must be null, got %#v", st.ProjectID)
	}
}
