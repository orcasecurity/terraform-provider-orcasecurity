package api_client

import (
	"fmt"
	"os"
	"testing"
)

// Needs TF_ACC, ORCASECURITY_API_ENDPOINT, ORCASECURITY_API_TOKEN and more than
// 20 custom rules in the org, so the last one sits past the API's default page.
func TestAccCustomSonarRuleExists_FindsRulePastDefaultPage(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC to run against a live tenant")
	}
	endpoint, token := os.Getenv("ORCASECURITY_API_ENDPOINT"), os.Getenv("ORCASECURITY_API_TOKEN")
	if endpoint == "" || token == "" {
		t.Skip("ORCASECURITY_API_ENDPOINT and ORCASECURITY_API_TOKEN are required")
	}
	c, err := NewAPIClient(&endpoint, &token)
	if err != nil {
		t.Fatal(err)
	}

	type pageType struct {
		Data []struct {
			RuleID string `json:"rule_id"`
		} `json:"data"`
		TotalItems int `json:"total_items"`
	}
	fetch := func(offset int) pageType {
		resp, err := c.Get(fmt.Sprintf("/api/sonar/rules/custom?limit=1&start_at_index=%d", offset))
		if err != nil {
			t.Fatal(err)
		}
		page := pageType{}
		if err := resp.ReadJSON(&page); err != nil {
			t.Fatal(err)
		}
		return page
	}

	total := fetch(0).TotalItems
	if total <= 20 {
		t.Skipf("org has %d custom rules; need more than 20 to exercise paging", total)
	}
	last := fetch(total - 1)
	if len(last.Data) != 1 {
		t.Fatalf("expected one rule at index %d, got %d", total-1, len(last.Data))
	}
	lastID := last.Data[0].RuleID

	found, err := c.customSonarRuleExists(lastID)
	if err != nil || !found {
		t.Fatalf("rule %s at index %d/%d not found: found=%v err=%v", lastID, total-1, total, found, err)
	}

	found, err = c.customSonarRuleExists("r0000000000")
	if err != nil || found {
		t.Fatalf("made-up id reported as existing: found=%v err=%v", found, err)
	}

	_, missing, err := c.getSonarRule("r0000000000")
	if err != nil || !missing {
		t.Fatalf("made-up id should be confirmed missing: missing=%v err=%v", missing, err)
	}
	t.Logf("total custom rules=%d, last index rule %s found, made-up id confirmed missing", total, lastID)
}
