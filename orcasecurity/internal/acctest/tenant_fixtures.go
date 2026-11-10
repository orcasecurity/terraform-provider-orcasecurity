package acctest

import (
	"net/url"
	"os"
	"strings"
	"testing"
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

const sonarTextDisabledMsg = "Sonar text rules are disabled"

// SkipIfSonarTextRulesDisabled skips when the org has the `ui_disable_sonar_text` flag on: the API
// then rejects every text `rule` with 400 "Sonar text rules are disabled ... use 'rule_json'". The
// probe query does not parse, so an org without the flag rejects it instead of creating a rule;
// if a rule is created anyway it is deleted.
func SkipIfSonarTextRulesDisabled(t *testing.T) {
	t.Helper()
	client := APIClient(t)
	resp, err := client.Post("/api/sonar/rules", map[string]any{
		"name": "tf-acc-sonar-text-probe", "rule": "tf acc probe (((", "category": "Best practices", "orca_score": 1,
	})
	if err == nil {
		var created struct {
			Data struct {
				ID string `json:"rule_id"`
			} `json:"data"`
		}
		if resp.ReadJSON(&created) == nil && created.Data.ID != "" {
			if delErr := client.DeleteCustomSonarAlert(created.Data.ID); delErr != nil {
				t.Fatalf("sonar probe rule %s was created and could not be deleted: %v", created.Data.ID, delErr)
			}
		}
		return
	}
	if strings.Contains(err.Error(), sonarTextDisabledMsg) {
		t.Skipf("org has the ui_disable_sonar_text flag on (%s); text sonar rules cannot be created", sonarTextDisabledMsg)
	}
}
