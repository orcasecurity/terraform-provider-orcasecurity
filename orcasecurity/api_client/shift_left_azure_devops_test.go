package api_client

import (
	"encoding/json"
	"testing"
)

func TestAzureDevopsAccount_UnmarshalLiveShape(t *testing.T) {
	// Fixture captured from GET /api/shiftleft/azure_devops/installations/{iid}/integrated_accounts/ (redacted).
	fixture := `{"id":"33333333-3333-3333-3333-333333333333","account_name":"acme-org","installation_mode":"SCAN_ALL_INCLUDE_FUTURE","default_policies":false,"policies":[{"id":"pol-1","name":"P1","type":"iac","builtin":true}],"configuration_settings":{"disable_scan_pull_requests":false,"comments_on_pull_requests":"ALWAYS","pr_summary_comment":"ALWAYS","skip_check_runs":"ALWAYS","config_file_support":"ENABLED","pr_summary_appendix":null,"installation_repositories_configuration":null}}`
	var acc AzureDevopsAccount
	if err := json.Unmarshal([]byte(fixture), &acc); err != nil {
		t.Fatal(err)
	}
	if acc.ID != "33333333-3333-3333-3333-333333333333" || acc.AccountName != "acme-org" {
		t.Errorf("bad id/account: %+v", acc)
	}
	if len(acc.Policies) != 1 || !acc.Policies[0].Builtin {
		t.Errorf("bad policies: %+v", acc.Policies)
	}
	if acc.ConfigSettings.SkipCheckRuns != "ALWAYS" {
		t.Errorf("bad config settings: %+v", acc.ConfigSettings)
	}
}
