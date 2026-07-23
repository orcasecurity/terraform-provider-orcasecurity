package api_client

import (
	"encoding/json"
	"testing"
)

func TestGitlabGroup_UnmarshalLiveShape(t *testing.T) {
	// Fixture captured from GET /api/shiftleft/gitlab/installations/{iid}/integrated_groups/ (redacted).
	fixture := `{"id":"22222222-2222-2222-2222-222222222222","account_name":"acme-group","installation_mode":"SCAN_ALL_INCLUDE_FUTURE","default_policies":false,"policies":[{"id":"pol-1","name":"P1","type":"iac","builtin":true}],"configuration_settings":{"disable_scan_pull_requests":false,"comments_on_pull_requests":"ALWAYS","pr_summary_comment":"ALWAYS","skip_check_runs":"ALWAYS","config_file_support":"ENABLED","pr_summary_appendix":null,"installation_repositories_configuration":null}}`
	var grp GitlabGroup
	if err := json.Unmarshal([]byte(fixture), &grp); err != nil {
		t.Fatal(err)
	}
	if grp.ID != "22222222-2222-2222-2222-222222222222" || grp.AccountName != "acme-group" {
		t.Errorf("bad id/account: %+v", grp)
	}
	if len(grp.Policies) != 1 || !grp.Policies[0].Builtin {
		t.Errorf("bad policies: %+v", grp.Policies)
	}
	if grp.ConfigSettings.SkipCheckRuns != "ALWAYS" {
		t.Errorf("bad config settings: %+v", grp.ConfigSettings)
	}
}
