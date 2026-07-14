package api_client

import (
	"fmt"
)

const scanConfigRulesBasePath = "/api/scan_configuration/rules"
const scanConfigBulkRulesPath = "/api/scan_configuration/bulk_rules"

// DataDetectionRuleTag is one tag selector of a rule. Rule tags are
// key/value selectors matched against asset tags — not plain strings.
type DataDetectionRuleTag struct {
	Keys   []string `json:"keys"`
	Values []string `json:"values"`
}

// DataDetectionRule is a scan configuration rule (feature "DSPM Scanning")
// from /api/scan_configuration/rules.
//
// The rules endpoint is non-standard REST:
//   - create: PUT on the collection (/rules), response carries data.rule_id
//   - update: POST /bulk_rules with {"rules_to_update": [{..., "rule_id": ...}]}
//     Bulk update is PARTIAL: keys absent from the payload keep their remote
//     value. The list fields below therefore have no omitempty — callers must
//     set them non-nil so clearing a list actually clears it server-side.
//   - there is NO PUT/PATCH on /rules/<rule_id>
type DataDetectionRule struct {
	ID                    string                 `json:"rule_id,omitempty"`
	OrganizationID        string                 `json:"organization,omitempty"`
	Name                  string                 `json:"rule_name"`
	Feature               string                 `json:"feature"`
	Action                string                 `json:"action"`
	Priority              *int64                 `json:"rule_priority,omitempty"`
	Enabled               bool                   `json:"is_enabled_rule"`
	SelectorCloudAccounts []string               `json:"selector_cloud_accounts"`
	SelectorBusinessUnits []string               `json:"selector_business_units"`
	Tags                  []DataDetectionRuleTag `json:"tags"`
	Policies              []string               `json:"policies"`
	IsDefaultRule         bool                   `json:"is_default_rule,omitempty"`
}

// GetDataDetectionRule retrieves one rule. Returns (nil, nil) on 404 so the
// resource Read can RemoveResource on remote drift.
// NOTE: unlike the list endpoint, retrieve responses carry the {status,data}
// envelope.
func (client *APIClient) GetDataDetectionRule(id string) (*DataDetectionRule, error) {
	resp, err := client.Get(fmt.Sprintf("%s/%s", scanConfigRulesBasePath, id))
	if resp != nil && resp.StatusCode() == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rule, err := readData[DataDetectionRule](resp)
	if err != nil {
		return nil, err
	}
	if rule.ID == "" {
		return nil, fmt.Errorf("rule retrieve: could not decode response: %s", string(resp.Body()))
	}
	return rule, nil
}

// CreateDataDetectionRule creates a rule via PUT on the collection
// (this is how the API works — not a mistake) and returns the new rule id.
func (client *APIClient) CreateDataDetectionRule(data DataDetectionRule) (string, error) {
	resp, err := client.Put(scanConfigRulesBasePath, data)
	if err != nil {
		return "", err
	}

	type createResult struct {
		RuleID string `json:"rule_id"`
	}
	result, err := readData[createResult](resp)
	if err != nil {
		return "", err
	}
	if result.RuleID == "" {
		return "", fmt.Errorf("rule create: missing rule_id in response: %s", string(resp.Body()))
	}
	return result.RuleID, nil
}

// UpdateDataDetectionRule updates a rule via POST /bulk_rules.
// data.ID must carry the rule_id of the rule being updated. Because bulk
// update only touches keys present in the payload, all mutable list fields
// are always serialized (see DataDetectionRule).
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
