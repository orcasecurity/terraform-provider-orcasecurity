package api_client

import (
	"encoding/json"
	"fmt"
)

const CustomTagRuleRuleTypeString = "string"
const CustomTagRuleRuleTypeJSON = "json"

type CustomTagRule struct {
	ID          string            `json:"id,omitempty"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Tags        map[string]string `json:"tags"`
	Rule        string            `json:"rule"`
	RuleType    string            `json:"rule_type"`
	Disabled    bool              `json:"disabled"`
}

// customTagRuleResponse mirrors the API response, where the rule field is a
// plain string when rule_type is "string" but a JSON object when rule_type is
// "json".
type customTagRuleResponse struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Tags        map[string]string `json:"tags"`
	Rule        json.RawMessage   `json:"rule"`
	RuleType    string            `json:"rule_type"`
	Disabled    bool              `json:"disabled"`
}

func (response customTagRuleResponse) toCustomTagRule() (*CustomTagRule, error) {
	var rule string
	if len(response.Rule) > 0 {
		if response.Rule[0] == '"' {
			if err := json.Unmarshal(response.Rule, &rule); err != nil {
				return nil, fmt.Errorf("failed to decode rule: %s", err.Error())
			}
		} else {
			rule = string(response.Rule)
		}
	}

	return &CustomTagRule{
		ID:          response.ID,
		Name:        response.Name,
		Description: response.Description,
		Tags:        response.Tags,
		Rule:        rule,
		RuleType:    response.RuleType,
		Disabled:    response.Disabled,
	}, nil
}

// customTagRuleRequest is the request payload for create/update operations.
// When rule_type is "json", the API expects the rule as a JSON object rather
// than a string, so Rule is typed as interface{}.
type customTagRuleRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Tags        map[string]string `json:"tags"`
	Rule        interface{}       `json:"rule"`
	RuleType    string            `json:"rule_type"`
	Disabled    bool              `json:"disabled"`
}

func newCustomTagRuleRequest(data CustomTagRule) (*customTagRuleRequest, error) {
	request := customTagRuleRequest{
		Name:        data.Name,
		Description: data.Description,
		Tags:        data.Tags,
		Rule:        data.Rule,
		RuleType:    data.RuleType,
		Disabled:    data.Disabled,
	}

	if data.RuleType == CustomTagRuleRuleTypeJSON {
		var ruleObject interface{}
		if err := json.Unmarshal([]byte(data.Rule), &ruleObject); err != nil {
			return nil, fmt.Errorf("rule must be valid JSON when rule_type is '%s': %s", CustomTagRuleRuleTypeJSON, err.Error())
		}
		request.Rule = ruleObject
	}

	return &request, nil
}

func (client *APIClient) DoesCustomTagRuleExist(id string) (bool, error) {
	resp, err := client.Get(fmt.Sprintf("/api/custom_tags/%s", id))
	if resp != nil && resp.StatusCode() == 404 {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (client *APIClient) GetCustomTagRule(id string) (*CustomTagRule, error) {
	type responseType struct {
		Data customTagRuleResponse `json:"data"`
	}

	resp, err := client.Get(fmt.Sprintf("/api/custom_tags/%s", id))
	if resp != nil && resp.StatusCode() == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	response := responseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return response.Data.toCustomTagRule()
}

func (client *APIClient) CreateCustomTagRule(data CustomTagRule) (*CustomTagRule, error) {
	type responseDataType struct {
		TagsRuleID string `json:"tags_rule_id"`
	}
	type responseType struct {
		Data responseDataType `json:"data"`
	}

	request, err := newCustomTagRuleRequest(data)
	if err != nil {
		return nil, err
	}

	resp, err := client.Post("/api/custom_tags", request)
	if err != nil {
		return nil, err
	}

	response := responseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	return client.GetCustomTagRule(response.Data.TagsRuleID)
}

func (client *APIClient) UpdateCustomTagRule(id string, data CustomTagRule) (*CustomTagRule, error) {
	type responseType struct {
		Data customTagRuleResponse `json:"data"`
	}

	request, err := newCustomTagRuleRequest(data)
	if err != nil {
		return nil, err
	}

	resp, err := client.Put(fmt.Sprintf("/api/custom_tags/%s", id), request)
	if err != nil {
		return nil, err
	}

	response := responseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return response.Data.toCustomTagRule()
}

func (client *APIClient) DeleteCustomTagRule(id string) error {
	_, err := client.Delete(fmt.Sprintf("/api/custom_tags/%s", id))
	return err
}
