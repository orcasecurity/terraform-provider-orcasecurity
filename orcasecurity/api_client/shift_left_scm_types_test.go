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
	if _, ok := cs["installation_repositories_configuration"]; ok {
		t.Errorf("expected installation_repositories_configuration to be omitted when nil, got: %s", raw)
	}
}
