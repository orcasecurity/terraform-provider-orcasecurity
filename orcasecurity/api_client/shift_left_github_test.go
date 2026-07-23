package api_client

import (
	"encoding/json"
	"testing"
)

func TestScmInstallationUpdate_MarshalShape(t *testing.T) {
	body := ScmInstallationUpdate{
		InstallationMode: "SCAN_ALL_INCLUDE_FUTURE",
		DefaultPolicies:  false,
		Policies:         []string{"pol-1", "pol-2"},
		ConfigSettings: ShiftLeftConfigSettings{
			DisableScanPullRequests: false,
			CommentsOnPullRequests:  "ALWAYS",
			PrSummaryComment:        "ONLY_ON_FAILED_ISSUES",
			SkipCheckRuns:           "ALWAYS",
			ConfigFileSupport:       "ENABLED",
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]interface{}
	_ = json.Unmarshal(raw, &got)
	for _, k := range []string{"installation_mode", "default_policies", "policies", "configuration_settings"} {
		if _, ok := got[k]; !ok {
			t.Errorf("missing top-level key %q in %s", k, raw)
		}
	}
	cs := got["configuration_settings"].(map[string]interface{})
	if cs["pr_summary_comment"] != "ONLY_ON_FAILED_ISSUES" {
		t.Errorf("configuration_settings.pr_summary_comment wrong: %v", cs["pr_summary_comment"])
	}
	for _, k := range []string{
		"disable_scan_pull_requests", "comments_on_pull_requests",
		"pr_summary_comment", "skip_check_runs", "config_file_support",
		"pr_summary_appendix",
	} {
		if _, ok := cs[k]; !ok {
			t.Errorf("missing required configuration_settings key %q in %s", k, raw)
		}
	}
}

func TestGithubInstallation_UnmarshalLiveShape(t *testing.T) {
	// Fixture captured from GET /api/shiftleft/github/installations/ (redacted).
	fixture := `{"id":"11111111-1111-1111-1111-111111111111","github_installation_id":123,"account_name":"acme","installation_mode":"SCAN_ALL_INCLUDE_FUTURE","default_policies":false,"policies":[{"id":"pol-1","name":"P1","type":"iac","builtin":true}],"configuration_settings":{"disable_scan_pull_requests":false,"comments_on_pull_requests":"ALWAYS","pr_summary_comment":"ALWAYS","skip_check_runs":"ALWAYS","config_file_support":"ENABLED","pr_summary_appendix":null,"installation_repositories_configuration":null}}`
	var inst GithubInstallation
	if err := json.Unmarshal([]byte(fixture), &inst); err != nil {
		t.Fatal(err)
	}
	if inst.ID != "11111111-1111-1111-1111-111111111111" || inst.AccountName != "acme" {
		t.Errorf("bad id/account: %+v", inst)
	}
	if len(inst.Policies) != 1 || !inst.Policies[0].Builtin {
		t.Errorf("bad policies: %+v", inst.Policies)
	}
	if inst.ConfigSettings.CommentsOnPullRequests != "ALWAYS" {
		t.Errorf("bad config settings: %+v", inst.ConfigSettings)
	}
}
