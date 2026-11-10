package acctest

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

// ExistingTemplateName returns envVar's value when set, otherwise a template_name that exists
// exactly once for serviceName on the tenant (the template data sources fail on duplicates).
// Skips when the tenant has none, so the test does not depend on one hardcoded fixture name.
func ExistingTemplateName(t *testing.T, serviceName, envVar string) string {
	t.Helper()
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	client := APIClient(t)
	resp, err := client.Get("/api/external_service/config?service_name=" + url.QueryEscape(serviceName))
	if err != nil {
		t.Fatalf("listing %s templates: %v", serviceName, err)
	}
	var body struct {
		Data []struct {
			TemplateName string `json:"template_name"`
		} `json:"data"`
	}
	if err := resp.ReadJSON(&body); err != nil {
		t.Fatalf("decoding %s templates: %v", serviceName, err)
	}
	count := map[string]int{}
	for _, d := range body.Data {
		count[d.TemplateName]++
	}
	for _, d := range body.Data {
		if d.TemplateName != "" && count[d.TemplateName] == 1 {
			return d.TemplateName
		}
	}
	t.Skipf("no uniquely named %s template on the tenant; create one or set %s", serviceName, envVar)
	return ""
}

const (
	sonarTextDisabledMsg = "Sonar text rules are disabled"
	sonarProbeName       = "tf-acc-sonar-text-probe"
)

// SkipIfSonarTextRulesDisabled skips when the org has the `ui_disable_sonar_text` flag on: the API
// then rejects every text `rule` with 400 "Sonar text rules are disabled ... use 'rule_json'". The
// probe query does not parse, so an org without the flag rejects it instead of creating a rule.
// It is sent with PostCreate (never replayed), and any rule left under the probe name is swept
// before and after, so a flaky gateway cannot leave a stray probe rule behind.
func SkipIfSonarTextRulesDisabled(t *testing.T) {
	t.Helper()
	client := APIClient(t)
	sweepSonarProbes(t, client)
	t.Cleanup(func() { sweepSonarProbes(t, client) })
	_, err := client.PostCreate("/api/sonar/rules", map[string]any{
		"name": sonarProbeName, "rule": "tf acc probe (((", "category": "Best practices", "orca_score": 1,
	})
	if err != nil && strings.Contains(err.Error(), sonarTextDisabledMsg) {
		t.Skipf("org has the ui_disable_sonar_text flag on (%s); text sonar rules cannot be created", sonarTextDisabledMsg)
	}
}

func sweepSonarProbes(t *testing.T, client *api_client.APIClient) {
	t.Helper()
	resp, err := client.Get("/api/sonar/rules/custom?limit=1000")
	if err != nil {
		t.Logf("listing custom sonar rules to sweep %s: %v", sonarProbeName, err)
		return
	}
	var page struct {
		Data []struct {
			ID   string `json:"rule_id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := resp.ReadJSON(&page); err != nil {
		t.Logf("decoding custom sonar rules: %v", err)
		return
	}
	for _, r := range page.Data {
		if r.Name == sonarProbeName && r.ID != "" {
			if err := client.DeleteCustomSonarAlert(r.ID); err != nil {
				t.Errorf("deleting stray sonar probe rule %s: %v", r.ID, err)
			}
		}
	}
}
