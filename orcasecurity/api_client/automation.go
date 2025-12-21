package api_client

import (
	"encoding/json"
	"fmt"
)

const AutomationAlertDismissalID = 1
const AutomationAlertScoreChangeID = 28
const AutomationAWSSecurityHubID = 37
const AutomationAwsSecurityLakeID = 25
const AutomationAwsSqsID = 33
const AutomationAzureDevopsID = 17
const AutomationAzureSentinelID = 7
const AutomationCoralogixID = 36
const AutomationEmailID = 5
const AutomationGcpPubSubID = 13
const AutomationGoogleSecurityOperationsSIEMID = 27
const AutomationJiraID = 10
const AutomationOpsgenieID = 4
const AutomationPagerDutyID = 3
const AutomationSnowflakeID = 26
const AutomationSplunkID = 8
const AutomationSumoLogicID = 6
const AutomationTinesID = 30
const AutomationTorqID = 16
const AutomationWebhookID = 12

// Covers both Slack v1 and Slack v2 integrations.
const AutomationSlackID = 2

type AutomationRange struct {
	Gte *string `json:"gte,omitempty"`
	Lte *string `json:"lte,omitempty"`
	Gt  *string `json:"gt,omitempty"`
	Lt  *string `json:"lt,omitempty"`
	Eq  *string `json:"eq,omitempty"`
}

type AutomationFilter struct {
	Field    string           `json:"field"`
	Includes []string         `json:"includes,omitempty"`
	Excludes []string         `json:"excludes,omitempty"`
	Range    *AutomationRange `json:"range,omitempty"`
}

type AutomationQuery struct {
	Filter []AutomationFilter `json:"filter"`
}

type AutomationAction struct {
	ID             string                 `json:"id,omitempty"`
	Type           int32                  `json:"type"`
	OrganizationID string                 `json:"organization,omitempty"`
	Data           map[string]interface{} `json:"data"`
}

func (a *AutomationAction) IsAlertDismissalTemplate() bool {
	return a.Type == AutomationAlertDismissalID
}

func (a *AutomationAction) IsAzureDevopsTemplate() bool {
	return a.Type == AutomationAzureDevopsID
}

func (a *AutomationAction) IsEmailTemplate() bool {
	return a.Type == AutomationEmailID
}

func (a *AutomationAction) IsJiraTemplate() bool {
	return a.Type == AutomationJiraID
}

func (a *AutomationAction) IsPagerDutyTemplate() bool {
	return a.Type == AutomationPagerDutyID
}

func (a *AutomationAction) IsSumoLogicTemplate() bool {
	return a.Type == AutomationSumoLogicID
}

func (a *AutomationAction) IsWebhookTemplate() bool {
	return a.Type == AutomationWebhookID
}

func (a *AutomationAction) IsSlackV2() bool {
	return a.Type == AutomationSlackID
}

type Automation struct {
	ID             string             `json:"id,omitempty"`
	Name           string             `json:"name"`
	BusinessUnits  []string           `json:"business_units"`
	Enabled        bool               `json:"is_enabled"`
	SonarQuery     map[string]int     `json:"sonar_query"`
	Description    string             `json:"description"`
	OrganizationID string             `json:"organization,omitempty"`
	Query          AutomationQuery    `json:"dsl_filter"`
	Actions        []AutomationAction `json:"actions"`
}

type automationAPIResponseType struct {
	Data Automation `json:"data"`
}

func (client *APIClient) GetAutomation(automationID string) (*Automation, error) {
	resp, err := client.Get(fmt.Sprintf("/api/rules/%s", automationID))
	if err != nil {
		return nil, err
	}

	if !resp.IsOk() {
		return nil, nil
	}

	response := automationAPIResponseType{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DoesAutomationExist(id string) (bool, error) {
	resp, _ := client.Head(fmt.Sprintf("/api/rules/%s", id))
	return resp.StatusCode() == 200, nil
}

func (client *APIClient) CreateAutomation(automation Automation) (*Automation, error) {
	resp, err := client.Post("/api/rules", automation)
	if err != nil {
		return nil, err
	}

	response := automationAPIResponseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil

}

func (client *APIClient) UpdateAutomation(ID string, data Automation) (*Automation, error) {
	resp, err := client.Put(fmt.Sprintf("/api/rules/%s", ID), data)
	if err != nil {
		return nil, err
	}

	response := automationAPIResponseType{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteAutomation(ID string) error {
	_, err := client.Delete(fmt.Sprintf("/api/rules/%s", ID))
	return err
}
