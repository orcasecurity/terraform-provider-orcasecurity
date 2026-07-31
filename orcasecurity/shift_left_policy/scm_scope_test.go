package shift_left_policy

import (
	"encoding/json"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestVerifyScmPostureScopeApplied_OK(t *testing.T) {
	plan := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("scm_posture"),
		ScmPosture: &scmPostureBlockModel{
			Scope: []scmScopeEntryModel{
				{Key: types.StringValue("github_installations"), Ids: []types.String{types.StringValue("org-1")}},
			},
		},
	}
	api := &api_client.ShiftLeftPolicy{
		Scope: json.RawMessage(`{"github_installations":[{"id":"org-1","name":"Org One"}],"gitlab_groups":[]}`),
	}
	if err := verifyScmPostureScopeApplied(plan, api); err != nil {
		t.Fatalf("expected scope to match, got %v", err)
	}
}

func TestVerifyScmPostureScopeApplied_MissingID(t *testing.T) {
	plan := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("scm_posture"),
		ScmPosture: &scmPostureBlockModel{
			Scope: []scmScopeEntryModel{
				{Key: types.StringValue("github_installations"), Ids: []types.String{
					types.StringValue("org-1"),
					types.StringValue("org-missing"),
				}},
			},
		},
	}
	api := &api_client.ShiftLeftPolicy{
		Scope: json.RawMessage(`{"github_installations":[{"id":"org-1","name":"Org One"}]}`),
	}
	err := verifyScmPostureScopeApplied(plan, api)
	if err == nil {
		t.Fatal("expected error when API drops a requested scope id")
	}
	if !strings.Contains(err.Error(), "github_installations=org-missing") {
		t.Fatalf("expected missing id in error, got %v", err)
	}
}

func TestVerifyScmPostureScopeApplied_EmptyAPIScope(t *testing.T) {
	plan := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("scm_posture"),
		ScmPosture: &scmPostureBlockModel{
			Scope: []scmScopeEntryModel{
				{Key: types.StringValue("github_installations"), Ids: []types.String{types.StringValue("gone")}},
			},
		},
	}
	api := &api_client.ShiftLeftPolicy{
		Scope: json.RawMessage(`{"github_installations":[],"gitlab_groups":[]}`),
	}
	if err := verifyScmPostureScopeApplied(plan, api); err == nil {
		t.Fatal("expected error when API returns empty scope lists")
	}
}

func TestVerifyScmPostureScopeApplied_SkipsNonScmAndBuiltin(t *testing.T) {
	iac := &shiftLeftPolicyResourceModel{Type: types.StringValue("iac")}
	if err := verifyScmPostureScopeApplied(iac, &api_client.ShiftLeftPolicy{}); err != nil {
		t.Fatalf("iac should skip: %v", err)
	}
	builtin := &shiftLeftPolicyResourceModel{
		Type:    types.StringValue("scm_posture"),
		Builtin: types.BoolValue(true),
		ScmPosture: &scmPostureBlockModel{
			Scope: []scmScopeEntryModel{
				{Key: types.StringValue("github_installations"), Ids: []types.String{types.StringValue("x")}},
			},
		},
	}
	if err := verifyScmPostureScopeApplied(builtin, &api_client.ShiftLeftPolicy{}); err != nil {
		t.Fatalf("builtin should skip: %v", err)
	}
}
