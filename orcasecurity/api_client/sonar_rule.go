package api_client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	deletedRuleRecheckDelay = 1 * time.Second
	// /api/sonar/rules/custom defaults to 20 per page; 1000 is its max_limit.
	customRulesPageSize      = 1000
	sonarRulesPath           = "/api/sonar/rules"
	customSonarRulesListPath = sonarRulesPath + "/custom"
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
	found := false
	err := client.scanCustomSonarRules(nil, func(rule customSonarRuleRow) bool {
		found = rule.RuleID == id
		return found
	})
	return found && err == nil, err
}

type customSonarRuleRow struct {
	RuleID string `json:"rule_id"`
	Name   string `json:"name"`
}

// scanCustomSonarRules pages the custom-rules list, calling visit on each rule until it returns true.
func (client *APIClient) scanCustomSonarRules(filters listFilters, visit func(customSonarRuleRow) bool) error {
	type pageType struct {
		Data       []customSonarRuleRow `json:"data"`
		TotalItems int                  `json:"total_items"`
	}
	offset := 0
	for {
		query := filters.query()
		query.Set("limit", strconv.Itoa(customRulesPageSize))
		query.Set("start_at_index", strconv.Itoa(offset))
		resp, err := client.Get(customSonarRulesListPath + "?" + query.Encode())
		if err != nil {
			return err
		}
		page := pageType{}
		if err := resp.ReadJSON(&page); err != nil {
			return err
		}
		for _, rule := range page.Data {
			if visit(rule) {
				return nil
			}
		}
		offset += len(page.Data)
		if len(page.Data) == 0 || offset >= page.TotalItems {
			return nil
		}
	}
}

// The list has no stable ordering and free_text_search is icontains over name, category, rule and creator.
func (client *APIClient) ListCustomSonarRuleIDsByName(name string) (map[string]bool, error) {
	ids := map[string]bool{}
	err := client.scanCustomSonarRules(listFilters{"free_text_search": strings.TrimSpace(name)}, func(rule customSonarRuleRow) bool {
		if sameName(rule.Name, name) {
			ids[rule.RuleID] = true
		}
		return false
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// Custom sonar and discovery alerts are both rules created through POST /api/sonar/rules.
func sonarRuleCreateOp[T any](client *APIClient, name string, body any, id func(*T) string, get func(string) (*T, error)) CreateOp[T] {
	return CreateOp[T]{
		What: fmt.Sprintf("custom alert named %q", name),
		ID:   id,
		Create: func() (*T, error) {
			resp, err := client.PostCreate(sonarRulesPath, body)
			if err != nil {
				return nil, err
			}
			return readData[T](resp)
		},
		IDs: func() (map[string]bool, error) { return client.ListCustomSonarRuleIDsByName(name) },
		Get: get,
	}
}

// SetCreatedCustomAlertRemediation sets a just-created custom alert's remediation text (keyed by its
// rule type), deleting the rule if that fails so no untracked rule is left behind.
func (client *APIClient) SetCreatedCustomAlertRemediation(ruleID, ruleType string, text *CustomSonarAlertRemediationText) error {
	if text == nil {
		return nil
	}
	withType := *text
	withType.AlertType = ruleType
	if err := client.setCustomRemediationText(withType); err != nil {
		return rollbackCreatedRule(err, ruleID, client.DeleteCustomSonarAlert)
	}
	return nil
}
