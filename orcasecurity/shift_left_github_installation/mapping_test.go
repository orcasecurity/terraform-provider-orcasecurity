package shift_left_github_installation

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestApiToState_MirrorsInstallationID(t *testing.T) {
	inst := &api_client.GithubInstallation{
		ID: "abc", AccountName: "acme", InstallationMode: "SCAN_ALL_INCLUDE_FUTURE",
		Policies: []api_client.ScmPolicyRef{{ID: "pol-1"}},
	}
	st := apiToState(inst)
	if st.ID.ValueString() != "abc" || st.InstallationID.ValueString() != "abc" {
		t.Errorf("id/installation_id mismatch: %+v", st)
	}
	elems := st.PoliciesIds.Elements()
	if len(elems) != 1 || elems[0].(types.String).ValueString() != "pol-1" {
		t.Errorf("policies_ids wrong: %+v", st.PoliciesIds)
	}
}
