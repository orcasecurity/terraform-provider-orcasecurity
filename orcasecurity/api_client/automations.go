package api_client

import (
	"encoding/json"
	"fmt"
)

const AutomationJiraActionID = 10
const AutomationSumoLogicID = 6
const AutomationAzureDevopsActionID = 17
const AutomationWebhookID = 12

type AutomationFilter struct {
	Field    string   `json:"field"`
	Includes []string `json:"includes,omitempty"`
	Excludes []string `json:"excludes,omitempty"`
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

func (a *AutomationAction) IsJiraIssue() bool {
	return a.Type == AutomationJiraActionID
}

func (a *AutomationAction) IsAzureDevopsWorkItem() bool {
	return a.Type == AutomationJiraActionID
}

func (a *AutomationAction) IsSumoLogic() bool {
	return a.Type == AutomationSumoLogicID
}

func (a *AutomationAction) IsWebhook() bool {
	return a.Type == AutomationWebhookID
}

type Automation struct {
	ID             string             `json:"id,omitempty"`
	Name           string             `json:"name"`
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

func (client *APIClient) IsAutomationExists(id string) (bool, error) {
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
