package api_client

import (
	"fmt"
)

const scanConfigRulesBasePath = "/api/scan_configuration/rules"
const scanConfigBulkRulesPath = "/api/scan_configuration/bulk_rules"

// DataDetectionRule is a scan configuration rule (feature "DSPM Scanning")
// from /api/scan_configuration/rules.
//
// The rules endpoint is non-standard REST:
//   - create: PUT on the collection (/rules), response carries data.rule_id
//   - update: POST /bulk_rules with {"rules_to_update": [{..., "rule_id": ...}]}
//   - list:   GET /rules returns a bare JSON array (no envelope)
//   - there is NO PUT/PATCH on /rules/<rule_id>
type DataDetectionRule struct {
	ID                    string   `json:"rule_id,omitempty"`
	OrganizationID        string   `json:"organization,omitempty"`
	Name                  string   `json:"rule_name"`
	Feature               string   `json:"feature"`
	Action                string   `json:"action"`
	Priority              *int64   `json:"rule_priority,omitempty"`
	Enabled               bool     `json:"is_enabled_rule"`
	SelectorCloudAccounts []string `json:"selector_cloud_accounts,omitempty"`
	SelectorBusinessUnits []string `json:"selector_business_units,omitempty"`
	Tags                  []string `json:"tags,omitempty"`
	Policies              []string `json:"policies,omitempty"`
	IsDefaultRule         bool     `json:"is_default_rule,omitempty"`
}

// GetDataDetectionRule retrieves one rule. Returns (nil, nil) on 404 so the
// resource Read can RemoveResource on remote drift. The decode is
// envelope-tolerant: it first tries {status,data} and falls back to a bare
// object (the rules list endpoint is known to skip the envelope, so retrieve
// is decoded defensively too).
func (client *APIClient) GetDataDetectionRule(id string) (*DataDetectionRule, error) {
	resp, err := client.Get(fmt.Sprintf("%s/%s", scanConfigRulesBasePath, id))
	if resp != nil && resp.StatusCode() == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	type responseType struct {
		Data DataDetectionRule `json:"data"`
	}
	response := responseType{}
	if err := resp.ReadJSON(&response); err == nil && response.Data.ID != "" {
		return &response.Data, nil
	}

	rule := DataDetectionRule{}
	if err := resp.ReadJSON(&rule); err != nil {
		return nil, err
	}
	if rule.ID == "" {
		return nil, fmt.Errorf("rule retrieve: could not decode response: %s", string(resp.Body()))
	}
	return &rule, nil
}

// ListDataDetectionRules lists all scan configuration rules.
// NOTE: the endpoint returns a bare JSON array, not the {status,data} envelope.
func (client *APIClient) ListDataDetectionRules() ([]DataDetectionRule, error) {
	resp, err := client.Get(scanConfigRulesBasePath)
	if err != nil {
		return nil, err
	}

	rules := []DataDetectionRule{}
	if err := resp.ReadJSON(&rules); err != nil {
		return nil, err
	}
	return rules, nil
}

// CreateDataDetectionRule creates a rule via PUT on the collection
// (this is how the API works — not a mistake) and returns the new rule id.
func (client *APIClient) CreateDataDetectionRule(data DataDetectionRule) (string, error) {
	resp, err := client.Put(scanConfigRulesBasePath, data)
	if err != nil {
		return "", err
	}

	type responseType struct {
		Data struct {
			RuleID string `json:"rule_id"`
		} `json:"data"`
	}
	response := responseType{}
	if err := resp.ReadJSON(&response); err != nil {
		return "", err
	}
	if response.Data.RuleID == "" {
		return "", fmt.Errorf("rule create: missing rule_id in response: %s", string(resp.Body()))
	}
	return response.Data.RuleID, nil
}

// UpdateDataDetectionRule updates a rule via POST /bulk_rules.
// data.ID must carry the rule_id of the rule being updated.
func (client *APIClient) UpdateDataDetectionRule(data DataDetectionRule) error {
	if data.ID == "" {
		return fmt.Errorf("rule update: rule_id is required")
	}
	payload := struct {
		RulesToUpdate []DataDetectionRule `json:"rules_to_update"`
	}{RulesToUpdate: []DataDetectionRule{data}}

	_, err := client.Post(scanConfigBulkRulesPath, payload)
	return err
}

func (client *APIClient) DeleteDataDetectionRule(id string) error {
	_, err := client.Delete(fmt.Sprintf("%s/%s", scanConfigRulesBasePath, id))
	return err
}
