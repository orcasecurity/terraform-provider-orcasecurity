package shift_left_gitlab_group

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestApiToState_MirrorsGroupID(t *testing.T) {
	inst := &api_client.GitlabGroup{
		ID: "abc", InstallationID: "inst-1", AccountName: "acme", InstallationMode: "SCAN_ALL_INCLUDE_FUTURE",
		Policies: []api_client.ScmPolicyRef{{ID: "pol-1"}},
	}
	st := apiToState(inst)
	if st.ID.ValueString() != "abc" || st.GroupID.ValueString() != "abc" {
		t.Errorf("id/group_id mismatch: %+v", st)
	}
	if st.InstallationID.ValueString() != "inst-1" {
		t.Errorf("installation_id mismatch: %+v", st)
	}
	elems := st.PoliciesIds.Elements()
	if len(elems) != 1 || elems[0].(types.String).ValueString() != "pol-1" {
		t.Errorf("policies_ids wrong: %+v", st.PoliciesIds)
	}
	if !st.ProjectID.IsNull() {
		t.Errorf("unbound project_id must be null, got %#v", st.ProjectID)
	}
}
