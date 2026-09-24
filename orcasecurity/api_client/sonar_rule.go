package api_client

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	deletedRuleRecheckDelay = 1 * time.Second
	// /api/sonar/rules/custom defaults to 20 per page; 1000 is its max_limit.
	customRulesPageSize = 1000
)

// getSonarRule fetches a custom sonar rule. The API has no 404 here: unknown ids
// return 400 and deleted rules 500, and that 500 is the same generic page as a
// real server error. A 500 is re-fetched once, and a rule is reported missing
// only after the custom-rules list confirms its id is gone, so a server error
// during refresh is never mistaken for a deleted alert and dropped from state.
func (client *APIClient) getSonarRule(id string) (resp *APIResponse, missing bool, err error) {
	path := fmt.Sprintf("/api/sonar/rules/%s", id)
	resp, err = client.Get(path)
	if resp != nil && resp.StatusCode() == http.StatusInternalServerError {
		if sleepErr := retrySleep(context.Background(), deletedRuleRecheckDelay+retryJitter(deletedRuleRecheckDelay/2)); sleepErr != nil {
			return nil, false, sleepErr
		}
		resp, err = client.Get(path)
	}
	if resp == nil || (resp.StatusCode() != http.StatusBadRequest && resp.StatusCode() != http.StatusInternalServerError) {
		return resp, false, err
	}

	exists, listErr := client.customSonarRuleExists(id)
	if listErr != nil {
		return resp, false, fmt.Errorf("rule %s returned HTTP %d and could not be confirmed as deleted: %w", id, resp.StatusCode(), listErr)
	}
	if exists {
		return resp, false, fmt.Errorf("rule %s still exists but GET returned HTTP %d: %w", id, resp.StatusCode(), err)
	}
	return resp, true, nil
}

func (client *APIClient) customSonarRuleExists(id string) (bool, error) {
	type pageType struct {
		Data []struct {
			RuleID string `json:"rule_id"`
		} `json:"data"`
		TotalItems int `json:"total_items"`
	}
	offset := 0
	for {
		resp, err := client.Get(fmt.Sprintf("/api/sonar/rules/custom?limit=%d&start_at_index=%d", customRulesPageSize, offset))
		if err != nil {
			return false, err
		}
		page := pageType{}
		if err := resp.ReadJSON(&page); err != nil {
			return false, err
		}
		for _, rule := range page.Data {
			if rule.RuleID == id {
				return true, nil
			}
		}
		offset += len(page.Data)
		if len(page.Data) == 0 || offset >= page.TotalItems {
			return false, nil
		}
	}
}
