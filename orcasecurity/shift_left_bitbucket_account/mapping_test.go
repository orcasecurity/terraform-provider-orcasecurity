package shift_left_bitbucket_account

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExpandUpdate_DefaultPoliciesClearsIds(t *testing.T) {
	m := &resourceModel{
		DefaultPolicies: types.BoolValue(true),
		PoliciesIds:     []types.String{types.StringValue("pol-1")},
	}
	body := expandUpdate(m)
	if len(body.Policies) != 0 {
		t.Errorf("expected empty policies when default_policies=true, got %v", body.Policies)
	}
	if !body.DefaultPolicies {
		t.Error("default_policies should be true")
	}
}

func TestExpandUpdate_ExplicitPolicies(t *testing.T) {
	m := &resourceModel{
		DefaultPolicies: types.BoolValue(false),
		PoliciesIds:     []types.String{types.StringValue("pol-1"), types.StringValue("pol-2")},
	}
	body := expandUpdate(m)
	if len(body.Policies) != 2 {
		t.Errorf("expected 2 policies, got %v", body.Policies)
	}
}

func TestApiToState_MirrorsAccountID(t *testing.T) {
	inst := &api_client.BitbucketAccount{
		ID: "abc", InstallationID: "inst-1", AccountName: "acme", InstallationMode: "SCAN_ALL_INCLUDE_FUTURE",
		Policies: []api_client.ScmPolicyRef{{ID: "pol-1"}},
	}
	st := apiToState(inst)
	if st.ID.ValueString() != "abc" || st.AccountID.ValueString() != "abc" {
		t.Errorf("id/account_id mismatch: %+v", st)
	}
	if st.InstallationID.ValueString() != "inst-1" {
		t.Errorf("installation_id mismatch: %+v", st)
	}
	if len(st.PoliciesIds) != 1 || st.PoliciesIds[0].ValueString() != "pol-1" {
		t.Errorf("policies_ids wrong: %+v", st.PoliciesIds)
	}
}
