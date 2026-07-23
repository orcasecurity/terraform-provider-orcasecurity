package api_client

import (
	"encoding/json"
	"testing"
)

func TestBitbucketAccount_UnmarshalLiveShape(t *testing.T) {
	// Fixture captured from GET /api/shiftleft/bitbucket/installations/{iid}/integrated_accounts/ (redacted).
	// Bitbucket's configuration_settings has no installation_repositories_configuration key at all
	// (unlike Azure DevOps/GitHub/GitLab, which return it as null) since Bitbucket's config type
	// doesn't model it.
	fixture := `{"id":"44444444-4444-4444-4444-444444444444","account_id":"acme-bb","account_name":"acme-bb","installation_mode":"SCAN_ALL_INCLUDE_FUTURE","default_policies":false,"policies":[{"id":"pol-1","name":"P1","type":"iac","builtin":true}],"configuration_settings":{"disable_scan_pull_requests":false,"comments_on_pull_requests":"ALWAYS","pr_summary_comment":"ALWAYS","skip_check_runs":"ALWAYS","config_file_support":"ENABLED","pr_summary_appendix":null}}`
	var acc BitbucketAccount
	if err := json.Unmarshal([]byte(fixture), &acc); err != nil {
		t.Fatal(err)
	}
	if acc.ID != "44444444-4444-4444-4444-444444444444" || acc.AccountName != "acme-bb" || acc.AccountID != "acme-bb" {
		t.Errorf("bad id/account: %+v", acc)
	}
	if len(acc.Policies) != 1 || !acc.Policies[0].Builtin {
		t.Errorf("bad policies: %+v", acc.Policies)
	}
	if acc.ConfigSettings.SkipCheckRuns != "ALWAYS" {
		t.Errorf("bad config settings: %+v", acc.ConfigSettings)
	}
	if acc.ConfigSettings.InstallationReposConfig != nil {
		t.Errorf("expected nil InstallationReposConfig for Bitbucket, got: %+v", acc.ConfigSettings.InstallationReposConfig)
	}
}

func TestScmInstallationID_Unmarshal(t *testing.T) {
	fixture := `{"id":"55555555-5555-5555-5555-555555555555"}`
	var inst scmInstallationID
	if err := json.Unmarshal([]byte(fixture), &inst); err != nil {
		t.Fatal(err)
	}
	if inst.ID != "55555555-5555-5555-5555-555555555555" {
		t.Errorf("bad installation id: %+v", inst)
	}
}

func TestScmInstallationUpdate_MarshalShape_Bitbucket(t *testing.T) {
	body := ScmInstallationUpdate{
		InstallationMode: "SCAN_ALL_INCLUDE_FUTURE",
		DefaultPolicies:  false,
		Policies:         []string{"pol-1", "pol-2"},
		ConfigSettings: ShiftLeftConfigSettings{
			DisableScanPullRequests: false,
			CommentsOnPullRequests:  "ALWAYS",
			PrSummaryComment:        "ONLY_ON_FAILED_SCAN",
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
	if cs["pr_summary_comment"] != "ONLY_ON_FAILED_SCAN" {
		t.Errorf("configuration_settings.pr_summary_comment wrong: %v", cs["pr_summary_comment"])
	}
	if _, ok := cs["installation_repositories_configuration"]; ok {
		t.Errorf("expected installation_repositories_configuration to be omitted, got: %s", raw)
	}
}
